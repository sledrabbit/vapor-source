import Fluent
import FluentPostgresDriver
import Foundation
import OpenAPIRuntime
import OpenAPIVapor
import SwiftDotenv
import Vapor

/// Configures the Vapor application.
///
/// Sets up database connections, migrations, middleware, and other services.
/// - Parameter app: The `Application` instance to configure.
public func configure(_ app: Application) async throws {
  do {
    try Dotenv.configure()
  } catch {
    app.logger.error("Unable to configure Dotenv: \(error)")
    throw error
  }

  let hostname = Environment.get("POSTGRES_HOST") ?? "localhost"
  let port =
    Environment.get("POSTGRES_PORT").flatMap(Int.init) ?? SQLPostgresConfiguration.ianaPortNumber
  let username = Environment.get("POSTGRES_USER")
  let password = Environment.get("POSTGRES_PASSWORD")
  let database = Environment.get("POSTGRES_DB")

  guard let username = username, let password = password, let database = database else {
    let missingVars = [
      username == nil ? "POSTGRES_USER" : nil,
      password == nil ? "POSTGRES_PASSWORD" : nil,
      database == nil ? "POSTGRES_DB" : nil,
    ].compactMap { $0 }.joined(separator: ", ")
    app.logger.critical("Missing required environment variables: \(missingVars)")
    throw Abort(
      .internalServerError, reason: "Missing required environment variables: \(missingVars)")
  }

  let dbConfig = SQLPostgresConfiguration(
    hostname: hostname,
    port: port,
    username: username,
    password: password,
    database: database,
    tls: .disable
  )
  app.databases.use(.postgres(configuration: dbConfig), as: .psql)

  app.migrations.add(CreateJob())
  app.migrations.add(CreateLanguages())
  app.migrations.add(CreateTechnologies())
  app.migrations.add(CreateJobLanguagePivot())
  app.migrations.add(CreateJobTechnologyPivot())

  try routes(app)

  app.middleware.use(FileMiddleware(publicDirectory: app.directory.publicDirectory))

}
