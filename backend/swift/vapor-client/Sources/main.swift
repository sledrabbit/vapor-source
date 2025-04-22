import Foundation
import OpenAPIRuntime
import OpenAPIURLSession
import SwiftSoup

let startTime = Date()
await scrapeJobs()
let executionTime = Date().timeIntervalSince(startTime)
print("Job scraping completed successfully in \(String(format: "%.2f", executionTime)) seconds")

func scrapeJobs() async {
  let testConfig = Config()
  let scraper = Scraper(config: testConfig)
  let query = "software engineer"

  print("Starting job scraping...")
  let jobs = scraper.scrapeJobs(query: query, config: testConfig)

  var count = 0
  for await job in jobs {
    count += 1
    print("\n--- Job \(count) ---")
    print("ID: \(job.jobId)")
    print("Title: \(job.title)")
    print("Company: \(job.company)")
    print("Location: \(job.location)")
    print("Posted: \(job.postedDate)")
    print("Salary: \(job.salary)")
    print("URL: \(job.url)")
  }
}

func testAPIClient() async throws {
  let job = Components.Schemas.Job(
    jobId: "job-123",
    title: "Swift Developer",
    company: "Vapor Inc.",
    location: "San Francisco, CA",
    postedDate: "2023-07-15",
    salary: "$120,000 - $150,000",
    url: "https://example.com/jobs/swift-developer",
    description: "We are looking for experienced Swift dev..."
  )

  let client = Client(
    serverURL: try Servers.Server2.url(),
    transport: URLSessionTransport()
  )

  let response = try await client.postJobs(body: .json(job))

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
