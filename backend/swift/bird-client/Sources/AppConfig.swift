import Foundation

struct AppConfig {
  let jobQuery: String
  let debugOutput: Bool
  let apiDryRun: Bool
  let useMockJobs: Bool
  let promptPath: String

  let openAIApiKey: String
  let openAIBaseURL: String
  let openAIModel: String

  let scraperMaxPages: Int
  let scraperBaseUrl: String
  let scraperRequestDelay: TimeInterval

  let parserMaxConcurrentTasks: Int

  init() throws {
    func getRequiredEnvVar(_ key: String) throws -> String {
      guard let value = ProcessInfo.processInfo.environment[key], !value.isEmpty else {
        throw ConfigurationError.missingEnvironmentVariable(key)
      }
      return value
    }

    func getEnvVar(_ key: String, default defaultValue: String) -> String {
      return ProcessInfo.processInfo.environment[key] ?? defaultValue
    }

    func getEnvVarAsBool(_ key: String, default defaultValue: Bool) -> Bool {
      let stringValue = ProcessInfo.processInfo.environment[key]?.lowercased() ?? ""
      return (stringValue == "true" || stringValue == "1") ? true : defaultValue
    }

    func getEnvVarAsInt(_ key: String, default defaultValue: Int) -> Int {
      return ProcessInfo.processInfo.environment[key].flatMap(Int.init) ?? defaultValue
    }

    func getEnvVarAsTimeInterval(_ key: String, default defaultValue: TimeInterval) -> TimeInterval
    {
      return ProcessInfo.processInfo.environment[key].flatMap(Double.init) ?? defaultValue
    }

    self.jobQuery = getEnvVar("QUERY", default: "software engineer")
    self.debugOutput = getEnvVarAsBool("DEBUG_OUTPUT", default: false)
    self.apiDryRun = getEnvVarAsBool("API_DRY_RUN", default: false)
    self.useMockJobs = getEnvVarAsBool("USE_MOCK_JOBS", default: false)
    self.promptPath = getEnvVar("LLM_PROMPT_PATH", default: "/var/task/prompt.txt")

    self.openAIApiKey = try getRequiredEnvVar("OPENAI_API_KEY")
    self.openAIBaseURL = getEnvVar(
      "OPENAI_BASE_URL", default: "https://api.openai.com/v1/chat/completions")
    self.openAIModel = getEnvVar("OPENAI_MODEL", default: "gpt-4.1-nano")

    self.scraperMaxPages = getEnvVarAsInt("SCRAPER_MAX_PAGES", default: 2)
    self.scraperBaseUrl = getEnvVar("SCRAPER_BASE_URL", default: "https://www.worksourcewa.com/")
    self.scraperRequestDelay = getEnvVarAsTimeInterval("SCRAPER_REQUEST_DELAY", default: 1.0)

    self.parserMaxConcurrentTasks = getEnvVarAsInt("PARSER_MAX_CONCURRENT_TASKS", default: 5)
  }
}

enum ConfigurationError: Error, LocalizedError {
  case missingEnvironmentVariable(String)

  var errorDescription: String? {
    switch self {
    case .missingEnvironmentVariable(let key):
      return "Missing required environment variable: \(key)"
    }
  }
}
