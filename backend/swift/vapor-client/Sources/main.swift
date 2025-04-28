import Foundation
import OpenAPIRuntime
import OpenAPIURLSession
import SwiftDotenv
import SwiftSoup

let debugEnabled = true
let devMode = true

let startTime = Date()
await scrapeJobs()
let executionTime = Date().timeIntervalSince(startTime)
debug("Job scraping completed successfully in \(String(format: "%.2f", executionTime)) seconds")

func scrapeJobs() async {
  let testConfig = Config()
  let scraper = Scraper(config: testConfig, debugEnabled: debugEnabled)

  do {
    try Dotenv.configure()
  } catch {
    print("Unable to configure Dotenv.")
    return
  }

  let query = Dotenv["QUERY"]?.stringValue ?? ""
  let promptPath = Dotenv["LLM_PROMPT_PATH"]?.stringValue ?? ""
  let testAPI = true

  debug("🔍 Starting job scraping...")
  let jobStream = scraper.scrapeJobs(query: query, config: testConfig)

  if !promptPath.isEmpty {
    do {
      let promptContent = try String(contentsOfFile: promptPath, encoding: .utf8)
      let parser = try Parser(
        jobStream: jobStream, prompt: promptContent, debugEnabled: debugEnabled, devMode: devMode)
      let processedJobStream = parser.parseJobs()

      for await job in processedJobStream {
        if testAPI {
          do {
            try await testAPIClient(job)
          } catch {
            print("Error sending job to API: \(error)")
          }
        }
      }
    } catch {
      print("Error with AI parsing: \(error)")
    }
  }
}

func testAPIClient(_ job: Job) async throws {
  if devMode {
    debug("\t📦 DEV MODE: Simulating server POST for job: \(job.title)")
    return
  }

  let client = Client(
    serverURL: try Servers.Server2.url(),
    transport: URLSessionTransport()
  )

  let response = try await client.postJobs(body: .json(job.toAPIModel()))

  switch response {
  case .created:
    debug("\t📦Post successful: \(job.title)")
  case .badRequest:
    print("❌ Bad request - invalid input provided")
  case .internalServerError:
    print("🔥 Server error encountered")
  case .undocumented(let statusCode, _):
    print("⚠️ Unexpected response with status code: \(statusCode)")
  }
}

func debug(_ message: String, isEnabled: Bool = true) {
  if isEnabled {
    print(message)
  }
}
