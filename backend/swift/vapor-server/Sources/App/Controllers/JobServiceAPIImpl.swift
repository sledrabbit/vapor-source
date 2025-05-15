import Fluent
import Foundation
import OpenAPIRuntime
import OpenAPIVapor
import Vapor

struct JobServiceAPIImpl: APIProtocol {

  private let app: Application

  init(app: Application) {
    self.app = app
  }

  func postJobs(
    _ input: Operations.PostJobs.Input
  ) async throws -> Operations.PostJobs.Output {
    guard case .json(let jobData) = input.body else {
      return .badRequest(.init())
    }

    if (try await findExistingJob(byId: jobData.jobId)) != nil {
      return .conflict(.init())
    }

    let job = try await createAndSaveJob(from: jobData)

    try await associateLanguages(for: job, from: jobData.languages)
    try await associateTechnologies(for: job, from: jobData.technologies)
    let responsePayload = Operations.PostJobs.Output.Created.Body.JsonPayload(title: job.title)

    return .created(.init(body: .json(responsePayload)))
  }

  private func findExistingJob(byId jobId: String) async throws -> Job? {
    return try await Job.query(on: self.app.db)
      .filter(\.$jobId == jobId)
      .first()
  }

  private func createAndSaveJob(from jobData: Components.Schemas.Job) async throws -> Job {
    let job = Job()
    job.jobId = jobData.jobId
    job.title = jobData.title
    job.company = jobData.company
    job.location = jobData.location
    job.postedDate = jobData.postedDate
    job.salary = jobData.salary.isEmpty ? "Not specified" : jobData.salary
    job.url = jobData.url
    job.description = jobData.description.isEmpty ? "Not specified" : jobData.description
    job.modality = jobData.modality?.rawValue
    job.expiresDate = jobData.expiresDate
    job.minYearsExperience = jobData.minYearsExperience
    job.minDegree = jobData.minDegree?.rawValue
    job.domain = jobData.domain?.rawValue
    job.parsedDescription = jobData.parsedDescription
    job.s3Pointer = jobData.s3Pointer

    try await job.save(on: self.app.db)
    return job
  }

  private func associateLanguages(
    for job: Job, from languages: [Components.Schemas.Language]?
  ) async throws {
    guard let languages = languages else { return }

    for languageData in languages {
      let language = try await findOrCreateLanguage(named: languageData.name)
      try await job.$languages.attach(language, on: self.app.db)
    }
  }

  private func findOrCreateLanguage(named name: String) async throws -> Language {
    let normalizedName = name.lowercased()
    if let existing = try await Language.query(on: self.app.db)
      .filter(\.$name == normalizedName)
      .first()
    {
      return existing
    }

    let language = Language(name: normalizedName)
    do {
      try await language.save(on: self.app.db)
      return language
    } catch {
      if let existing = try await Language.query(on: self.app.db)
        .filter(\.$name == normalizedName)
        .first()
      {
        return existing
      }
      throw error
    }
  }

  private func associateTechnologies(
    for job: Job, from technologies: [Components.Schemas.Technology]?
  ) async throws {
    guard let technologies = technologies else { return }

    for techData in technologies {
      let technology = try await findOrCreateTechnology(named: techData.name)
      try await job.$technologies.attach(technology, on: self.app.db)
    }
  }

  private func findOrCreateTechnology(named name: String) async throws -> Technology {
    let normalizedName = name.lowercased()
    if let existing = try await Technology.query(on: self.app.db)
      .filter(\.$name == normalizedName)
      .first()
    {
      return existing
    }

    let technology = Technology(name: normalizedName)
    do {
      try await technology.save(on: self.app.db)
      return technology
    } catch {
      if let existing = try await Technology.query(on: self.app.db)
        .filter(\.$name == normalizedName)
        .first()
      {
        return existing
      }
      throw error
    }
  }
}
