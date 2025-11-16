import Foundation

enum MockData {
  static let jobs: [Job] = [
    Job(
      id: 1234,
      jobId: "mock-1",
      title: "Senior Swift Engineer",
      company: "Umbrella Corp",
      location: "Silicon Valley",
      modality: "In-person",
      postedDate: "2025-11-16",
      expiresDate: nil,
      salary: "$140,000 - $170,000",
      url: "https://example.com/jobs/mock-1",
      minYearsExperience: 5,
      minDegree: "Bachelor's",
      domain: "Backend",
      description: "Build and maintain backend services in Swift using Vapor.",
      parsedDescription: "Write Swift code",
      s3Pointer: nil,
      languages: [Language(id: nil, name: "Swift")],
      technologies: [
        Technology(id: nil, name: "Vapor"),
        Technology(id: nil, name: "PostgreSQL"),
        Technology(id: nil, name: "Vapor"),
      ]
    )
  ]
}
