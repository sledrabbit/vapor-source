import Foundation

struct Job: Codable {
  var id: UInt?
  var jobId: String
  var title: String
  var company: String
  var location: String
  var modality: String?
  var postedDate: String
  var expiresDate: String?
  var salary: String
  var url: String
  var minYearsExperience: Int?
  var minDegree: String?
  var domain: String?
  var description: String
  var parsedDescription: String?
  var s3Pointer: String?
  var languages: [Language]?
  var technologies: [Technology]?
}

struct Language: Codable {
  var id: UInt?
  var name: String
}

struct Technology: Codable {
  var id: UInt?
  var name: String
}

struct Constants {
  static let allowedDomains = [
    "Backend",
    "Full-Stack",
    "AI/ML",
    "Data",
    "QA",
    "Front-End",
    "Security",
    "DevOps",
    "Mobile",
    "Site Reliability",
    "Networking",
    "Embedded Systems",
    "Gaming",
    "Financial",
    "Other",
  ]

  static let allowedModalities = [
    "In-Office",
    "Hybrid",
    "Remote",
  ]

  static let allowedDegrees = [
    "Bachelor's",
    "Master's",
    "Ph.D",
    "Unspecified",
  ]
}
