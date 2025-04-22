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

  func parseJob(stream: AsyncStream<Job>) async {
    for await job in stream {
      let finalPrompt = "\(prompt)\n\nJob description: \(job.description)"

      do {
        let response = try await messenger.sendMessage(
          prompt: finalPrompt, content: job.description)

        if let content = response.content {
          print("\n===== AI Parsing for Job: \(job.title) =====")
          print(content)
          print("==========================================\n")
        } else {
          print("Empty response received from OpenAI")
        }

      } catch {
        print("API call failure: \(error.localizedDescription)")
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
