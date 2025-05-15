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

struct APILanguage: Codable {
  var id: Int?
  var name: String
}

struct APITechnology: Codable {
  var id: Int?
  var name: String
}

struct APIJob: Codable {
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
  var languages: [APILanguage]?
  var technologies: [APITechnology]?
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

extension Job {
  func toAPIModel() -> APIJob {
    return APIJob(
      jobId: self.jobId,
      title: self.title,
      company: self.company,
      location: self.location,
      modality: self.modality,
      postedDate: self.postedDate,
      expiresDate: self.expiresDate,
      salary: self.salary,
      url: self.url,
      minYearsExperience: self.minYearsExperience,
      minDegree: self.minDegree,
      domain: self.domain,
      description: self.description,
      parsedDescription: self.parsedDescription,
      s3Pointer: self.s3Pointer,
      languages: self.languages?.map { lang in
        APILanguage(id: lang.id.flatMap { Int($0) }, name: lang.name)
      },
      technologies: self.technologies?.map { tech in
        APITechnology(id: tech.id.flatMap { Int($0) }, name: tech.name)
      }
    )
  }
}
