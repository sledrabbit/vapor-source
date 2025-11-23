import AWSSDKIdentity
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
    let jobStream = makeJobStream()

    debug(
      "\t🔧 DynamoDB config - region: \(config.awsRegion ?? "nil"), endpoint: \(config.dynamoDBEndPoint ?? "nil")"
    )

    do {
      let table = try await DynamoDBTable(
        region: config.awsRegion,
        tableName: "Jobs",
        endpoint: config.dynamoDBEndPoint
      )
      let parser = Parser(config: config, logger: logger)
      let processor = JobProcessor(
        parser: parser,
        table: table,
        config: config,
        logger: logger
      )
      await processor.run(jobStream: jobStream)
    } catch {
      logger.error("Failed to initialize DynamoDB table: \(error)")
      return
    }

    let executionTime = Date().timeIntervalSince(startTime)
    debug("Job processing completed in \(String(format: "%.2f", executionTime)) seconds")
  }

  private func debug(_ message: String, isEnabled: Bool = true) {
    if isEnabled {
      print(message)
    }
  }

  private func makeJobStream() -> AsyncStream<Job> {
    if config.useMockJobs {
      logger.info("\t🧪 MOCK MODE: Using static mock jobs instead of scraping")
      return AsyncStream { continuation in
        for job in MockData.jobs {
          continuation.yield(job)
        }
        continuation.finish()
      }
    }
    return scraper.scrapeJobs(query: config.jobQuery)
  }

}
