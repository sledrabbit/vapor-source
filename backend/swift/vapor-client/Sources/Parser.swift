import Foundation
import SwiftDotenv

enum ParserError: Error {
  case missingAPIKey
}

struct Parser {
  var jobStream: AsyncStream<Job>
  let prompt: String
  private let messenger: OpenAIMessenger

  init(jobStream: AsyncStream<Job>, prompt: String) throws {
    self.jobStream = jobStream
    self.prompt = prompt

    guard let apiKey = Dotenv["OPENAI_API_KEY"] else {
      throw ParserError.missingAPIKey
    }

    self.messenger = OpenAIMessenger(apiKey: apiKey.stringValue)
  }

  func parseJobs(maxConcurrent: Int = 25) -> AsyncStream<Job> {
    return AsyncStream { continuation in
      Task {
        await withTaskGroup(of: Job?.self) { group in
          var runningTasks = 0

          for await job in jobStream {
            if runningTasks >= maxConcurrent {
              if let completedJob = await group.next() {
                if let job = completedJob {
                  continuation.yield(job)
                }
                runningTasks -= 1
              }
            }

            group.addTask { [self] in
              let finalPrompt = "\(prompt)\n\nJob description: \(job.description)"
              do {
                let response = try await self.messenger.sendMessage(
                  prompt: finalPrompt, content: job.description)
                if let content = response.content {
                  print("\n===== AI Parsing for Job: \(job.title) =====")
                  print("\t\t \(job.company)")
                  print(content)
                  print(job.url)
                  print("==========================================\n")
                  return job
                } else {
                  print("Empty response received from OpenAI")
                  return nil
                }
              } catch {
                print("API call failure: \(error.localizedDescription)")
                return nil
              }
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
}

struct OpenAIMessenger {
  private let apiKey: String
  private let baseURL = "https://api.openai.com/v1/chat/completions"

  init(apiKey: String) {
    self.apiKey = apiKey
  }

  struct ChatCompletionResponse: Decodable {
    let choices: [Choice]

    var content: String? {
      choices.first?.message.content
    }

    struct Choice: Decodable {
      let message: Message
    }

    struct Message: Decodable {
      let content: String
    }
  }

  func sendMessage(prompt: String, content: String) async throws -> ChatCompletionResponse {
    let fullMessage = prompt + " " + content

    guard let url = URL(string: baseURL) else {
      throw NSError(domain: "Invalid URL", code: 400)
    }

    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.addValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
    request.addValue("application/json", forHTTPHeaderField: "Content-Type")

    let requestBody: [String: Any] = [
      "model": "gpt-4o-mini",
      "messages": [
        ["role": "user", "content": fullMessage]
      ],
    ]

    let jsonData = try JSONSerialization.data(withJSONObject: requestBody)
    request.httpBody = jsonData

    let (data, response) = try await URLSession.shared.data(for: request)

    guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
      throw NSError(
        domain: "HTTP Error", code: (response as? HTTPURLResponse)?.statusCode ?? 500)
    }

    return try JSONDecoder().decode(ChatCompletionResponse.self, from: data)
  }
}
