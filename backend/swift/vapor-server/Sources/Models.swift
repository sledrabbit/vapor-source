struct Job: Codable {
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

struct Language: Codable {
    let id: UInt?
    let name: String
}

struct Technology: Codable {
    let id: UInt?
    let name: String
}
