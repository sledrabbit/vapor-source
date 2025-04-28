import Fluent
import FluentPostgresDriver
import Foundation
import OpenAPIRuntime
import OpenAPIVapor
import SwiftDotenv
import Vapor

struct JobServiceAPIImpl: APIProtocol {

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
        return try await Job.query(on: app.db)
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

        try await job.save(on: app.db)
        return job
    }

    private func associateLanguages(
        for job: Job, from languages: [Components.Schemas.Language]?
    ) async throws {
        guard let languages = languages else { return }

        for languageData in languages {
            let language = try await findOrCreateLanguage(named: languageData.name)
            try await job.$languages.attach(language, on: app.db)
        }
    }

    private func findOrCreateLanguage(named name: String) async throws -> Language {
        let language =
            try await Language.query(on: app.db)
            .filter(\.$name == name)
            .first()
            .flatMap { $0 } ?? Language(name: name)

        if language.id == nil {
            try await language.save(on: app.db)
        }

        return language
    }

    private func associateTechnologies(
        for job: Job, from technologies: [Components.Schemas.Technology]?
    ) async throws {
        guard let technologies = technologies else { return }

        for techData in technologies {
            let technology = try await findOrCreateTechnology(named: techData.name)
            try await job.$technologies.attach(technology, on: app.db)
        }
    }

    private func findOrCreateTechnology(named name: String) async throws -> Technology {
        let technology =
            try await Technology.query(on: app.db)
            .filter(\.$name == name)
            .first()
            .flatMap { $0 } ?? Technology(name: name)

        if technology.id == nil {
            try await technology.save(on: app.db)
        }

        return technology
    }
}

// create Vapor app
let app: Application = try await Vapor.Application.make()

// get env variables
do {
    try Dotenv.configure()
} catch {
    print("Unable to configure Dotenv.")
    throw error
}

let hostname = Dotenv["POSTGRES_HOST"]?.stringValue ?? ""
let port =
    Int(Dotenv["POSTGRES_PORT"]?.stringValue ?? "") ?? SQLPostgresConfiguration.ianaPortNumber
let username = Dotenv["POSTGRES_USER"]?.stringValue ?? ""
let password = Dotenv["POSTGRES_PASSWORD"]?.stringValue ?? ""
let database = Dotenv["POSTGRES_DB"]?.stringValue ?? ""

// db connection
let dbConfig = SQLPostgresConfiguration(
    hostname: hostname,
    port: port,
    username: username,
    password: password,
    database: database,
    tls: .disable
)
app.databases.use(.postgres(configuration: dbConfig), as: .psql)

// db migrations
app.migrations.add(CreateJob())
app.migrations.add(CreateLanguages())
app.migrations.add(CreateTechnologies())
app.migrations.add(CreateJobLanguagePivot())
app.migrations.add(CreateJobTechnologyPivot())

// _ = app.autoMigrate()
// app.logger.info("Database migrations completed.")

// create VaporTransport using app
let transport: VaporTransport = VaporTransport(routesBuilder: app)

// handler type that conforms the generated protocol
let handler: JobServiceAPIImpl = JobServiceAPIImpl()

// call generated fucntion on your impl to add its request handlers to app
try handler.registerHandlers(on: transport, serverURL: Servers.Server1.url())

// Add Vapor middleware to serve the contents of the Public/ directory.
app.middleware.use(FileMiddleware(publicDirectory: app.directory.publicDirectory))

app.get("openapi") { $0.redirect(to: "/openapi.html", redirectType: .permanent) }

try await app.execute()
