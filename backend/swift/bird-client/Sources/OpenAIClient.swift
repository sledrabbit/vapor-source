import Foundation

#if canImport(FoundationNetworking)
  import FoundationNetworking
#endif

struct OpenAIClient: AIMessaging {
  private let config: AppConfig

  init(config: AppConfig) {
    self.config = config
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
  func sendMessage(content: String) async throws -> AIResponse {
    let request = try makeRequest(with: content)
    let response = try await performRequest(request)

    return AIResponse(content: response.content)
  }

  private func makeRequest(with message: String) throws -> URLRequest {
    guard let url = URL(string: config.openAIBaseURL) else {
      throw NSError(
        domain: "Invalid URL",
        code: 400,
        userInfo: [NSLocalizedDescriptionKey: "Invalid OpenAI Base URL: \(config.openAIBaseURL)"])
    }

    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    request.addValue("Bearer \(config.openAIApiKey)", forHTTPHeaderField: "Authorization")
    request.addValue("application/json", forHTTPHeaderField: "Content-Type")
    
    let responseFormat: [String: Any] = [
      "type": "json_schema",
      "json_schema": Job.schema
    ]

    let requestBody: [String: Any] = [
      "model": config.openAIModel,
      "messages": [
        ["role": "user", "content": message]
      ],
      "response_format": responseFormat,
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
