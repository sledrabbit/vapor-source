import Foundation
import Logging
import OpenAPIRuntime
import OpenAPIURLSession

struct AppConfig {
  let query: String
  let promptPath: String
  let debugOutput: Bool
  let apiDryRun: Bool
  let openAIApiKey: String
}

class AppRunner {
  let config: AppConfig
  let scraper: Scraper
  let logger: Logger

  init(config: AppConfig) throws {
    self.config = config
    let scraperConfig = Config()
    self.logger = Logger(label: "app.runner")
    self.scraper = Scraper(
      config: scraperConfig, debugOutput: config.debugOutput, logger: self.logger)
  }

  func run() async {
    let startTime = Date()

    let scrapedJobStream = await scrapeJobs()

    let parser: Parser
    do {

      let promptContent = try String(contentsOfFile: config.promptPath, encoding: .utf8)
      parser = try Parser(
        jobStream: scrapedJobStream,
        prompt: promptContent,
        apiKey: config.openAIApiKey,
        apiDryRun: config.apiDryRun,
        logger: logger
      )
    } catch {
      logger.error("Error initializing Parser or reading prompt: \(error)")
      return
    }

    await parser.parseAndPost()

    let executionTime = Date().timeIntervalSince(startTime)
    debug("Job processing completed in \(String(format: "%.2f", executionTime)) seconds")
  }

  private func scrapeJobs() async -> AsyncStream<Job> {
    return scraper.scrapeJobs(query: config.query, config: scraper.config)
  }

  private func debug(_ message: String, isEnabled: Bool = true) {
    if isEnabled {
      print(message)
    }
  }

}
