import Foundation
import OpenAPIRuntime
import OpenAPIURLSession

struct AppConfig {
  let query: String
  let promptPath: String
  let debugOutput: Bool
  let apiDryRun: Bool
}

class AppRunner {
  let config: AppConfig
  let scraper: Scraper

  init(config: AppConfig) throws {
    self.config = config
    let scraperConfig = Config()
    self.scraper = Scraper(config: scraperConfig, debugOutput: config.debugOutput)
  }

  func run() async {
    let startTime = Date()

    let scrapedJobStream = await scrapeJobs()

    let parsedJobStream = await parseJobs(from: scrapedJobStream)

    await postJobs(from: parsedJobStream)

    let executionTime = Date().timeIntervalSince(startTime)
    debug("Job processing completed in \(String(format: "%.2f", executionTime)) seconds")
  }

  private func scrapeJobs() async -> AsyncStream<Job> {
    return scraper.scrapeJobs(query: config.query, config: scraper.config)
  }

  private func parseJobs(from jobStream: AsyncStream<Job>) async -> AsyncStream<Job> {
    do {
      let promptContent = try String(contentsOfFile: config.promptPath, encoding: .utf8)
      let parser = try Parser(
        jobStream: jobStream,
        prompt: promptContent,
        debugOutput: config.debugOutput,
        apiDryRun: config.apiDryRun)
      return parser.parseJobs()
    } catch {
      print("...")
      return AsyncStream { continuation in continuation.finish() }
    }
  }

  private func postJobs(from jobStream: AsyncStream<Job>) async {
    for await job in jobStream {
      do {
        if config.apiDryRun {
          debug("\t🧪 DEV MODE: Simulating server POST for job: \(job.title)")
          continue
        }

        let client = Client(
          serverURL: try Servers.Server2.url(),
          transport: URLSessionTransport()
        )

        let response = try await client.postJobs(body: .json(job.toAPIModel()))

        switch response {
        case .created:
          debug("\t📦Post successful: \(job.title)")
        case .conflict:
          debug("\t🟡 Duplicate job (skipped): \(job.title)")
        case .badRequest:
          print("❌ Bad request - invalid input provided")
        case .internalServerError:
          print("🔥 Server error encountered")
        case .undocumented(let statusCode, _):
          print("⚠️ Unexpected response with status code: \(statusCode)")
        }
      } catch {
        print("Error sending job to API: \(error)")
      }
    }
  }

  private func debug(_ message: String, isEnabled: Bool = true) {
    if isEnabled {
      print(message)
    }
  }

}
