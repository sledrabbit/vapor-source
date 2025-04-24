import Foundation

struct OpenAIClient {
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
}

extension OpenAIClient {
  func sendMessage(prompt: String, content: String) async throws -> ChatCompletionResponse {
    let fullMessage = prompt + " " + content

    let request = try makeRequest(with: fullMessage)
    return try await performRequest(request)
  }

  private func makeRequest(with message: String) throws -> URLRequest {
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
        ["role": "user", "content": message]
      ],
      "response_format": ["type": "json_object"],
    ]

    request.httpBody = try JSONSerialization.data(withJSONObject: requestBody)
    return request
  }

  private func performRequest(_ request: URLRequest) async throws -> ChatCompletionResponse {
    let (data, response) = try await URLSession.shared.data(for: request)

    guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
      throw NSError(
        domain: "HTTP Error",
        code: (response as? HTTPURLResponse)?.statusCode ?? 500
      )
    }

    return try JSONDecoder().decode(ChatCompletionResponse.self, from: data)
  }
}
