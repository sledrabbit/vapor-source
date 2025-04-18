import Foundation
import OpenAPIRuntime
import OpenAPIURLSession

// test job
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
