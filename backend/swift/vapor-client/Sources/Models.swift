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

extension Job {
  func toAPIModel() -> Components.Schemas.Job {
    return Components.Schemas.Job(
      jobId: self.jobId,
      title: self.title,
      company: self.company,
      location: self.location,
      modality: self.modality.flatMap { Components.Schemas.Job.ModalityPayload(rawValue: $0) },
      postedDate: self.postedDate,
      expiresDate: self.expiresDate,
      salary: self.salary,
      url: self.url,
      minYearsExperience: self.minYearsExperience,
      minDegree: self.minDegree.flatMap { Components.Schemas.Job.MinDegreePayload(rawValue: $0) },
      domain: self.domain.flatMap { Components.Schemas.Job.DomainPayload(rawValue: $0) },
      description: self.description,
      parsedDescription: self.parsedDescription,
      s3Pointer: self.s3Pointer,
      languages: self.languages?.map { lang in
        Components.Schemas.Language(id: lang.id.flatMap { Int($0) }, name: lang.name)
      },
      technologies: self.technologies?.map { tech in
        Components.Schemas.Technology(id: tech.id.flatMap { Int($0) }, name: tech.name)
      }
    )
  }
}
