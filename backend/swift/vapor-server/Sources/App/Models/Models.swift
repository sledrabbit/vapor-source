import Fluent
import Vapor

struct JobLocal: Codable {
    let jobId: String
    let title: String
    let company: String
    let location: String
    let postedDate: String
    let salary: String
    let url: String
    let description: String

    let modality: String?
    let expiresDate: String?
    let minYearsExperience: Int?
    let minDegree: String?
    let domain: String?
    let parsedDescription: String?
    let s3Pointer: String?
    let languages: [Language]?
    let technologies: [Technology]?
}

struct LanguageLocal: Codable {
    let id: UInt?
    let name: String
}

struct TechnologyLocal: Codable {
    let id: UInt?
    let name: String
}

// Make Language conform to Fluent's Model and Content
final class Language: Model, Content, @unchecked Sendable {
    static let schema = "languages"

    @ID(key: .id)
    var id: UUID?

    @Field(key: "name")
    var name: String

    @Siblings(through: JobLanguagePivot.self, from: \.$language, to: \.$job)
    var jobs: [Job]

    init() {}

    init(id: UUID? = nil, name: String) {
        self.id = id
        self.name = name.lowercased()
    }
}

// Make Technology conform to Fluent's Model and Content
final class Technology: Model, Content, @unchecked Sendable {
    static let schema = "technologies"

    @ID(key: .id)
    var id: UUID?

    @Field(key: "name")
    var name: String

    @Siblings(through: JobTechnologyPivot.self, from: \.$technology, to: \.$job)
    var jobs: [Job]

    init() {}

    init(id: UUID? = nil, name: String) {
        self.id = id
        self.name = name.lowercased()
    }
}

// Pivot table for the Job <-> Language relationship
final class JobLanguagePivot: Model, @unchecked Sendable {
    static let schema = "job_language_pivot"

    @ID(key: .id)
    var id: UUID?

    @Parent(key: "job_id")
    var job: Job

    @Parent(key: "language_id")
    var language: Language

    init() {}

    init(id: UUID? = nil, job: Job, language: Language) throws {
        self.id = id
        self.$job.id = try job.requireID()
        self.$language.id = try language.requireID()
    }
}

// Pivot table for the Job <-> Technology relationship
final class JobTechnologyPivot: Model, @unchecked Sendable {
    static let schema = "job_technology_pivot"

    @ID(key: .id)
    var id: UUID?

    @Parent(key: "job_id")
    var job: Job

    @Parent(key: "technology_id")
    var technology: Technology

    init() {}

    init(id: UUID? = nil, job: Job, technology: Technology) throws {
        self.id = id
        self.$job.id = try job.requireID()
        self.$technology.id = try technology.requireID()
    }
}

// Make Job conform to Fluent's Model and Content (for Vapor routes)
final class Job: Model, Content, @unchecked Sendable {
    static let schema = "jobs"  // Database table name

    @ID(key: .id)
    var id: UUID?

    @Field(key: "job_id")
    var jobId: String

    @Field(key: "title")
    var title: String

    @Field(key: "company")
    var company: String

    @Field(key: "location")
    var location: String

    @Field(key: "posted_date")
    var postedDate: String

    @Field(key: "salary")
    var salary: String

    @Field(key: "url")
    var url: String

    @Field(key: "description")
    var description: String

    @OptionalField(key: "modality")
    var modality: String?

    @OptionalField(key: "expires_date")
    var expiresDate: String?

    @OptionalField(key: "min_years_experience")
    var minYearsExperience: Int?

    @OptionalField(key: "min_degree")
    var minDegree: String?

    @OptionalField(key: "domain")
    var domain: String?

    @OptionalField(key: "parsed_description")
    var parsedDescription: String?

    @OptionalField(key: "s3_pointer")
    var s3Pointer: String?

    @Siblings(through: JobLanguagePivot.self, from: \.$job, to: \.$language)
    var languages: [Language]

    @Siblings(through: JobTechnologyPivot.self, from: \.$job, to: \.$technology)
    var technologies: [Technology]

    init() {}

    init(
        id: UUID? = nil, jobId: String, title: String, company: String, location: String,
        postedDate: String, salary: String, url: String, description: String,
        modality: String? = nil, expiresDate: String? = nil, minYearsExperience: Int? = nil,
        minDegree: String? = nil, domain: String? = nil, parsedDescription: String? = nil,
        s3Pointer: String? = nil
    ) {
        self.id = id
        self.jobId = jobId
        self.title = title
        self.company = company
        self.location = location
        self.postedDate = postedDate
        self.salary = salary
        self.url = url
        self.description = description
        self.modality = modality
        self.expiresDate = expiresDate
        self.minYearsExperience = minYearsExperience
        self.minDegree = minDegree
        self.domain = domain
        self.parsedDescription = parsedDescription
        self.s3Pointer = s3Pointer
    }
}
