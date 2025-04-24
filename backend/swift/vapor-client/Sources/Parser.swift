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

  init(jobStream: AsyncStream<Job>, prompt: String) throws {
    self.jobStream = jobStream
    self.prompt = prompt

    guard let apiKey = Dotenv["OPENAI_API_KEY"] else {
      throw ParserError.missingAPIKey
    }

    self.messenger = OpenAIClient(apiKey: apiKey.stringValue)
  }
}

extension Parser {
  func parseJobs(maxConcurrent: Int = 25) -> AsyncStream<Job> {
    return AsyncStream { continuation in
      Task {
        await withTaskGroup(of: Job?.self) { group in
          var runningTasks = 0

          for await job in jobStream {
            if runningTasks >= maxConcurrent {
              if let completedJob = await group.next(), let job = completedJob {
                continuation.yield(job)
              }
              runningTasks -= 1
            }

            group.addTask {
              await processJob(job)
            }
            runningTasks += 1
          }
          for await result in group {
            if let job = result {
              continuation.yield(job)
            }
          }
        }
        continuation.finish()
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
    let finalPrompt = "\(prompt)\n\nJob description: \(job.description)"

    do {
      print("🤖 Analyzing job: \(job.title)")
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
