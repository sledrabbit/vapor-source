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
    let jobStream: AsyncStream<Job>

    if config.useMockJobs {
      logger.info("\t🧪 MOCK MODE: Using static mock jobs instead of scraping")
      jobStream = AsyncStream { continuation in
        for job in MockData.jobs {
          continuation.yield(job)
        }
        continuation.finish()
      }
    } else {
      jobStream = scraper.scrapeJobs(query: config.jobQuery)
    }

    let parser = Parser(
      jobStream: jobStream,
      config: config,
      logger: logger
    )

    debug(
      "\t🔧 DynamoDB config - region: \(config.awsRegion ?? "nil"), endpoint: \(config.dynamoDBEndPoint ?? "nil")"
    )

    do {
      let table = try await DynamoDBTable(
        region: config.awsRegion,
        tableName: "Jobs",
        endpoint: config.dynamoDBEndPoint
      )
      for await parsedJob in parser.parseJobs() {
        await table.postJob(parsedJob, config: config, logger: logger)
      }
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

}
