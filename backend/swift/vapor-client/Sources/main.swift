import Foundation
import OpenAPIRuntime
import OpenAPIURLSession
import SwiftDotenv
import SwiftSoup

let startTime = Date()
await scrapeJobs()
let executionTime = Date().timeIntervalSince(startTime)
print("Job scraping completed successfully in \(String(format: "%.2f", executionTime)) seconds")

func scrapeJobs() async {
  let testConfig = Config()
  let scraper = Scraper(config: testConfig)

  do {
    try Dotenv.configure()
  } catch {
    print("Unable to configure Dotenv.")
    return
  }

  let query = Dotenv["QUERY"]?.stringValue ?? ""
  let promptPath = Dotenv["LLM_PROMPT_PATH"]?.stringValue ?? ""
  let testAPI = true

  print("Starting job scraping...")
  let jobStream = scraper.scrapeJobs(query: query, config: testConfig)

  if !promptPath.isEmpty {
    do {
      let promptContent = try String(contentsOfFile: promptPath, encoding: .utf8)
      let parser = try Parser(jobStream: jobStream, prompt: promptContent)
      let processedJobStream = parser.parseJobs()

      var count = 0
      for await job in processedJobStream {
        count += 1
        print("\n--- Job \(count) ---")
        print("ID: \(job.jobId)")
        print("Title: \(job.title)")
        print("Company: \(job.company)")
        print("Location: \(job.location)")
        print("Posted: \(job.postedDate)")
        print("Salary: \(job.salary)")
        print("URL: \(job.url)")

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
  let client = Client(
    serverURL: try Servers.Server2.url(),
    transport: URLSessionTransport()
  )

  let response = try await client.postJobs(body: .json(job.toAPIModel()))

  switch response {
  case .created(let createdResponse):
    print("Job created successfully!")
    switch createdResponse.body {
    case .json(let createdJob):
      print("Job ID: \(createdJob.jobId)")
      print("Title: \(createdJob.title)")
      print("Company: \(createdJob.company)")
    }
  case .badRequest:
    print("Bad request - invalid input provided")
  case .internalServerError:
    print("Server error encountered")
  case .undocumented(let statusCode, _):
    print("Unexpected response with status code: \(statusCode)")
  }
}
