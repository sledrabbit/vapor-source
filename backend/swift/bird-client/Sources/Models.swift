import Foundation

struct Job: Codable, Sendable {
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

struct Language: Codable, Sendable {
  var id: UInt?
  var name: String
}

struct Technology: Codable, Sendable {
  var id: UInt?
  var name: String
}

struct APILanguage: Codable, Sendable {
  var id: Int?
  var name: String
}

struct APITechnology: Codable, Sendable {
  var id: Int?
  var name: String
}

struct APIJob: Codable, Sendable {
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

  nonisolated(unsafe) static let schema: [String: Any] = [
    "name": "OpenAIJobParsingResponse",
    "strict": true,
    "schema": [
      "type": "object",
      "properties": [
        "ParsedDescription": [
          "type": "string",
          "description": "A concise summary of the job role and key responsibilities",
        ],
        "DeadlineDate": [
          "type": "string",
          "description":
            "Deadline or expiry date for the job posting. Use 'Ongoing until requisition is closed' if not specified",
        ],
        "MinDegree": [
          "type": "string",
          "enum": ["Bachelor's", "Master's", "Ph.D", "Unspecified"],
          "description": "Minimum degree required for the role",
        ],
        "MinYearsExperience": [
          "type": "integer",
          "minimum": 0,
          "maximum": 25,
          "description": """
          Minimum years of professional experience required. CRITICAL RULES:
          1) If job title contains 'Senior' or 'Sr.' set to at least 4 years,
          2) If job title contains 'Principal', 'Staff', 'Lead', or 'Director' set to at least 7 years,
          3) If job title contains 'Mid-level' set to at least 2 years,
          4) Otherwise extract specific years from description,
          5) If no experience mentioned and no seniority keywords, set to 0
          """,
        ],
        "Modality": [
          "type": "string",
          "enum": ["Remote", "Hybrid", "In-Office"],
          "description": "Work arrangement. Default to 'In-Office' if unclear",
        ],
        "Domain": [
          "type": "string",
          "enum": [
            "Backend", "Full-Stack", "AI/ML", "Data", "QA", "Front-End", "Security",
            "DevOps", "Mobile", "Site Reliability", "Networking", "Embedded Systems",
            "Gaming", "Financial", "Other",
          ],
          "description":
            "Technical domain. If description focuses on server-side or microservices development, choose 'Backend'",
        ],
        "Languages": [
          "type": "array",
          "items": ["type": "string"],
          "description":
            "Programming languages mentioned in the job. Only include programming languages, not spoken languages",
        ],
        "Technologies": [
          "type": "array",
          "items": ["type": "string"],
          "description":
            "Software tools, frameworks, databases, and technologies mentioned in the job",
        ],
        "IsSoftwareEngineerRelated": [
          "type": "boolean",
          "description": "Whether the job is primarily related to software engineering",
        ],
      ],
      "required": [
        "ParsedDescription",
        "DeadlineDate",
        "MinDegree",
        "MinYearsExperience",
        "Modality",
        "Domain",
        "Languages",
        "Technologies",
        "IsSoftwareEngineerRelated",
      ],
      "additionalProperties": false,
    ],
  ]
}
