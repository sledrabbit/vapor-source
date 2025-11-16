import Foundation
import Logging

#if canImport(FoundationNetworking)
  import FoundationNetworking
#endif

// MARK: - Dependencies

protocol AIMessaging: Sendable {
  func sendMessage(prompt: String, content: String) async throws -> AIResponse
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

  init(
    jobStream: AsyncStream<Job>,
    prompt: String,
    config: AppConfig,
    logger: Logger,
    aiMessenger: AIMessaging? = nil
  ) {
    self.jobStream = jobStream
    self.prompt = prompt
    self.config = config
    self.logger = logger
    self.aiMessenger = aiMessenger ?? OpenAIClient(config: config)
  }
}

// MARK: - Public API

extension Parser {
  func parseJobs() async -> [Job] {
    let limiter = ConcurrencyLimiter(limit: config.parserMaxConcurrentTasks)

    return await withTaskGroup(of: Job?.self) { group -> [Job] in
      for await job in jobStream {
        await limiter.wait()

        group.addTask {
          defer {
            Task { await limiter.signal() }
          }

          do {
            if let processedJob = try await self.processJob(job) {
              return processedJob
            } else {
              self.debug("\t🦉 Filtering out or failed to parse job: \(job.title)")
              return nil
            }
          } catch {
            self.logger.error("Error processing job \(job.jobId): \(error)")
            return nil
          }
        }
      }
      var parsedJobs: [Job] = []
      while let result = await group.next() {
        if let job = result {
          parsedJobs.append(job)
        }
      }
      return parsedJobs
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

// MARK: - Utility Functions

extension Parser {
  private func debug(_ message: String) {
    if config.debugOutput {
      logger.info("\(message)")
    }
  }
}
