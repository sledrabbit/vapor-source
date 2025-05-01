import Foundation
import Logging
import OpenAPIURLSession
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
  let debugOutput: Bool
  let apiDryRun: Bool
  let logger: Logger

  init(
    jobStream: AsyncStream<Job>,
    prompt: String,
    apiKey: String,
    debugOutput: Bool = true,
    apiDryRun: Bool,
    logger: Logger
  )
    throws
  {
    self.jobStream = jobStream
    self.prompt = prompt
    self.debugOutput = debugOutput
    self.apiDryRun = apiDryRun
    self.messenger = OpenAIClient(apiKey: apiKey)
    self.logger = logger
  }
  private func debug(_ message: String) {
    if debugOutput {
      logger.info("\(message)")
    }
  }
}

extension Parser {
  func parseAndPost(maxConcurrentTasks: Int = 5) async {
    let limiter = ConcurrencyLimiter(limit: maxConcurrentTasks)

    await withTaskGroup(of: Void.self) { group in
      for await job in jobStream {
        await limiter.wait()

        group.addTask {
          defer {
            Task { await limiter.signal() }
          }
          var processedJob: Job? = nil

          do {
            processedJob = try await self.processJob(job)

            if let jobToPost = processedJob {
              await self.postSingleJob(jobToPost)
            } else {
              debug("\t owl ....")
            }
          } catch {
            logger.error("error")
          }
        }
      }
    }
  }

  private func postSingleJob(_ job: Job) async {
    if apiDryRun {
      debug("\t🧪 DEV MODE: Simulating server POST for job: \(job.title)")
      return
    }
    do {
      let client = Client(
        serverURL: try Servers.Server2.url(),
        transport: URLSessionTransport()
      )

      let response = try await client.postJobs(body: .json(job.toAPIModel()))

      switch response {
      case .created:
        debug("\t📦Post successful: \(job.title)")
      case .conflict:
        debug("\t🟡 Duplicate job (skipped): \(job.title)")
      case .badRequest:
        print("❌ Bad request - invalid input provided")
      case .internalServerError:
        print("🔥 Server error encountered")
      case .undocumented(let statusCode, _):
        print("⚠️ Unexpected response with status code: \(statusCode)")
      }
    } catch {
      print("Error sending job to API: \(error)")
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

      return updatedJob
    } catch {
      logger.error("JSON Parsing error: \(error)")
      logger.error("Failed to parse JSON: \(content)")
      return nil
    }
  }

  private func processJob(_ job: Job) async throws -> Job? {
    if apiDryRun {
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
        logger.warning("⚠️ Empty response received from OpenAI")
        return nil
      }

      logger.debug("Raw content for job \(job.title): \(content)")

      return try await parseAIResponse(content: content, originalJob: job)
    } catch {
      logger.error("❌ API call failure: \(error.localizedDescription)")
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
          logger.error(
            "❌ Max retry attempts (\(maxAttempts)) reached. Operation failed. Error: \(error)")
          throw error
        }
        let jitter = Double.random(in: -jitterFactor...jitterFactor) * currentDelay
        let delayWithJitter = max(0, currentDelay + jitter)
        let delayInSeconds = String(format: "%.2f", delayWithJitter)
        logger.warning(
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
