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
  func parseJobs() -> AsyncStream<Job> {
    return AsyncStream { continuation in
      let processingTask = Task {
        await withTaskGroup(of: Void.self) { group in
          for await job in jobStream {
            group.addTask {
              let processedJob = await self.processJob(job)

              if let processedJob = processedJob {
                continuation.yield(processedJob)
              }
            }
          }
        }
        continuation.finish()
      }
      continuation.onTermination = { _ in
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

  private func processJob(_ job: Job) async -> Job? {
    if devMode {
      debug("\t🧪 DEV MODE: Simulating AI response for job: \(job.title)")
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
      let response = try await messenger.sendMessage(prompt: finalPrompt, content: job.description)
      guard let content = response.content else {
        print("⚠️ Empty response received from OpenAI")
        return nil
      }

      return try await parseAIResponse(content: content, originalJob: job)
    } catch {
      print("❌ API call failure: \(error.localizedDescription)")
      return nil
    }
  }
}
