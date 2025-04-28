import Foundation
import SwiftDotenv

enum ParserError: Error {
  case missingAPIKey
  case emptyResponse
  case jsonParsingError(Error)
  case apiError(Error)
}

struct AIParseResponse: Decodable {
  let ParsedDescription: String
  let MinDegree: String
  let MinYearsExperience: Int
  let Modality: String
  let Domain: String
  let Languages: [Language]
  let Technologies: [Technology]
  let IsSoftwareEngineerRelated: Bool
}

struct Parser {
  var jobStream: AsyncStream<Job>
  let prompt: String
  private let messenger: OpenAIClient
  let debugEnabled: Bool
  let devMode: Bool

  init(jobStream: AsyncStream<Job>, prompt: String, debugEnabled: Bool = true, devMode: Bool) throws
  {
    self.jobStream = jobStream
    self.prompt = prompt
    self.debugEnabled = debugEnabled
    self.devMode = devMode

    guard let apiKey = Dotenv["OPENAI_API_KEY"] else {
      throw ParserError.missingAPIKey
    }

    self.messenger = OpenAIClient(apiKey: apiKey.stringValue)
  }
}

extension Parser {
  func parseJobs(maxConcurrentTasks: Int = 5) -> AsyncStream<Job> {
    let limiter = ConcurrencyLimiter(limit: maxConcurrentTasks)

    return AsyncStream { continuation in
      let processingTask = Task {
        await withTaskGroup(of: Void.self) { group in
          for await job in jobStream {
            await limiter.wait()

            group.addTask {
              defer {
                Task { await limiter.signal() }
              }
              do {
                if let processedJob = try await self.processJob(job) {
                  continuation.yield(processedJob)
                } else {
                  debug("\t🦉 Filtering out or failed to parse job: \(job.title)")
                }
              } catch {
                print(
                  "Error processing job \(job.id != nil ? String(job.id!) : job.title): \(error)")
              }
            }
          }
        }
        continuation.finish()
      }
      continuation.onTermination = { @Sendable _ in
        processingTask.cancel()
      }
    }
  }

  private func parseAIResponse(content: String, originalJob: Job) async throws -> Job? {
    var updatedJob = originalJob
    let jsonData = Data(content.utf8)
    let decoder = JSONDecoder()
    decoder.keyDecodingStrategy = .convertFromSnakeCase

    do {
      let parsedFields: AIParseResponse = try decoder.decode(AIParseResponse.self, from: jsonData)

      guard parsedFields.IsSoftwareEngineerRelated else {
        debug(
          "\t🦉 Filtering out non-software related job (based on AI response): \(originalJob.title)")
        return nil
      }

      updatedJob.parsedDescription = parsedFields.ParsedDescription
      updatedJob.minDegree = parsedFields.MinDegree
      updatedJob.minYearsExperience = parsedFields.MinYearsExperience
      updatedJob.modality = parsedFields.Modality
      updatedJob.domain = parsedFields.Domain
      updatedJob.languages = parsedFields.Languages
      updatedJob.technologies = parsedFields.Technologies

      // print("Successfully parsed updatedJob: \(updatedJob.title)")
      return updatedJob
    } catch {
      print("JSON Parsing error: \(error)")
      print("Failed to parse JSON: \(content)")
      return nil
    }
  }

  private func processJob(_ job: Job) async throws -> Job? {
    if devMode {
      debug("\t🧪 DEV MODE: Simulating AI response for job: \(job.title)")

      if false {
        debug("\t🦉 DEV MODE: Filtering out non-software related job: \(job.title)")
        return nil
      }

      var updatedJob = job
      updatedJob.parsedDescription = "Mock parsed description for \(job.title)"
      updatedJob.minDegree = "Bachelor's"
      updatedJob.minYearsExperience = 3
      updatedJob.modality = "Remote"
      updatedJob.domain = "Software Development"
      updatedJob.languages = [Language(name: "Swift")]
      updatedJob.technologies = [Technology(name: "Vapor")]

      return updatedJob
    }

    let finalPrompt = "\(prompt)\n\nJob description: \(job.description)"

    do {
      debug("\t🤖 Analyzing job: \(job.title)")
      let response = try await retryWithBackoff {
        try await messenger.sendMessage(prompt: finalPrompt, content: job.description)
      }
      guard let content = response.content else {
        print("⚠️ Empty response received from OpenAI")
        return nil
      }

      return try await parseAIResponse(content: content, originalJob: job)
    } catch {
      print("❌ API call failure: \(error.localizedDescription)")
      throw error
    }
  }

  private func retryWithBackoff<T>(
    maxAttempts: Int = 10,
    initialDelay: TimeInterval = 1.0,
    backoffFactor: Double = 2.0,
    jitterFactor: Double = 0.1,
    operation: @escaping () async throws -> T
  ) async throws -> T {
    var attempts = 0
    var currentDelay = initialDelay

    while attempts < maxAttempts {
      attempts += 1
      do {
        return try await operation()
      } catch {
        if attempts == maxAttempts {
          print("❌ Max retry attempts (\(maxAttempts)) reached. Operation failed. Error: \(error)")
          throw error
        }
        let jitter = Double.random(in: -jitterFactor...jitterFactor) * currentDelay
        let delayWithJitter = max(0, currentDelay + jitter)
        let delayInSeconds = String(format: "%.2f", delayWithJitter)
        print(
          "⚠️ Attempt \(attempts)/\(maxAttempts) failed. Retrying in \(delayInSeconds)s... Error: \(error)"
        )
        try await Task.sleep(nanoseconds: UInt64(currentDelay * 1_000_000_000))
        currentDelay *= backoffFactor
      }
    }
    fatalError("Retry logic exited loop unexpectedly.")
  }
}

actor ConcurrencyLimiter {
  private let limit: Int
  private var currentCount = 0
  private var waitQueue: [CheckedContinuation<Void, Never>] = []

  init(limit: Int) {
    self.limit = limit
  }

  func wait() async {
    await withCheckedContinuation { continuation in
      if currentCount < limit {
        currentCount += 1
        continuation.resume()
      } else {
        waitQueue.append(continuation)
      }
    }
  }

  func signal() async {
    if let continuation = waitQueue.first {
      waitQueue.removeFirst()
      continuation.resume()
    } else {
      currentCount -= 1
    }
  }
}
