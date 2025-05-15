import Foundation
import Logging

#if canImport(FoundationNetworking)
  import FoundationNetworking
#endif

// MARK: - Dependencies

protocol AIMessaging: Sendable {
  func sendMessage(prompt: String, content: String) async throws -> AIResponse
}

protocol JobPosting: Sendable {
  func postJob(_ job: APIJob, to url: URL) async throws -> (Int, Data?)
}

extension URLSession: JobPosting {
  func postJob(_ job: APIJob, to url: URL) async throws -> (Int, Data?) {
    let jsonData = try JSONEncoder().encode(job)

    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    request.httpBody = jsonData

    let (data, response) = try await data(for: request)

    guard let httpResponse = response as? HTTPURLResponse else {
      throw URLError(.badServerResponse)
    }

    return (httpResponse.statusCode, data)
  }
}

struct AIResponse: Decodable {
  let content: String?
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

struct Parser: Sendable {
  let jobStream: AsyncStream<Job>
  let prompt: String
  let config: AppConfig
  let logger: Logger
  let aiMessenger: AIMessaging
  let jobPoster: JobPosting

  init(
    jobStream: AsyncStream<Job>,
    prompt: String,
    config: AppConfig,
    logger: Logger,
    aiMessenger: AIMessaging? = nil,
    jobPoster: JobPosting? = nil
  ) {
    self.jobStream = jobStream
    self.prompt = prompt
    self.config = config
    self.logger = logger
    self.aiMessenger = aiMessenger ?? OpenAIClient(config: config)
    self.jobPoster = jobPoster ?? URLSession.shared
  }
}

// MARK: - Public API

extension Parser {
  func parseAndPost() async {
    let limiter = ConcurrencyLimiter(limit: config.parserMaxConcurrentTasks)

    await withTaskGroup(of: Void.self) { group in
      for await job in jobStream {
        await limiter.wait()

        group.addTask {
          defer {
            Task { await limiter.signal() }
          }

          do {
            if let processedJob = try await self.processJob(job) {
              await self.postSingleJob(processedJob)
            } else {
              self.debug("\t🦉 Filtering out or failed to parse job: \(job.title)")
            }
          } catch {
            self.logger.error("Error processing job \(job.jobId): \(error)")
          }
        }
      }
      await group.waitForAll()
    }
  }
}

// MARK: - Job Processing

extension Parser {
  private func processJob(_ job: Job) async throws -> Job? {
    if config.apiDryRun {
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
      let response = try await retryWithBackoff(logger: logger) {
        try await aiMessenger.sendMessage(prompt: finalPrompt, content: job.description)
      }

      guard let content = response.content else {
        logger.warning("⚠️ Empty response received from OpenAI for job \(job.jobId)")
        return nil
      }

      logger.debug("Raw AI content for job \(job.jobId): \(content)")

      return try await parseAIResponse(content: content, originalJob: job)
    } catch {
      logger.error("❌ AI API call failure for job \(job.jobId): \(error.localizedDescription)")
      throw error
    }
  }

  private func parseAIResponse(content: String, originalJob: Job) async throws -> Job? {
    var updatedJob = originalJob
    let jsonData = Data(content.utf8)
    let decoder = JSONDecoder()
    decoder.keyDecodingStrategy = .convertFromSnakeCase

    do {
      let parsedFields = try decoder.decode(AIParseResponse.self, from: jsonData)

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
}

// MARK: - Job Posting

extension Parser {
  private func postSingleJob(_ job: Job) async {
    if config.apiDryRun {
      debug("\t🧪 DEV MODE: Simulating server POST for job: \(job.title)")
      return
    }

    do {
      guard let serverUrl = URL(string: config.apiServerURL) else {
        logger.error("❌ Invalid API Server URL: \(config.apiServerURL)")
        return
      }

      let apiJobPayload = job.toAPIModel()

      let (statusCode, _) = try await jobPoster.postJob(apiJobPayload, to: serverUrl)

      switch statusCode {
      case 201:
        debug("\t📦 Post successful (201 Created): \(job.title)")
      case 200:
        debug("\t📦 Post successful (200 OK): \(job.title)")
      case 409:
        debug("\t🟡 Duplicate job (409 Conflict - skipped): \(job.title)")
      case 400:
        logger.error(
          "❌ Bad request (400) for job \(job.jobId) - invalid input provided. Check payload.")
      case 500...599:
        logger.error("🔥 Server error (\(statusCode)) for job \(job.jobId).")
      default:
        logger.warning("⚠️ Unexpected response status code: \(statusCode) for job \(job.jobId)")
      }
    } catch {
      logger.error("Error sending job \(job.jobId) to API: \(error)")
    }
  }
}

// MARK: - Utility Functions

extension Parser {
  private func debug(_ message: String) {
    if config.debugOutput {
      logger.info("\(message)")
    }
  }
}
