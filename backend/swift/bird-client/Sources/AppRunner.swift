import Foundation
import Logging

class AppRunner {
  let config: AppConfig
  let scraper: Scraper
  let logger: Logger

  init(config: AppConfig, logger: Logger) throws {
    self.config = config
    self.logger = Logger(label: "app.runner")
    self.scraper = Scraper(config: config, logger: self.logger)
  }

  func run() async {
    let startTime = Date()

    let scrapedJobStream = scraper.scrapeJobs(query: config.jobQuery)

    let parser: Parser
    do {

      let promptContent = try String(contentsOfFile: config.promptPath, encoding: .utf8)
      parser = Parser(
        jobStream: scrapedJobStream,
        prompt: promptContent,
        config: config,
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

  private func debug(_ message: String, isEnabled: Bool = true) {
    if isEnabled {
      print(message)
    }
  }

}
