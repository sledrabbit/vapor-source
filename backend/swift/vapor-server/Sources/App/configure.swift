import AWSClientRuntime
import AWSSSM
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
    app.logger.info("Dotenv configuration failed or .env file not found: \(error)")
  }

  // --- Database Configuration ---
  let hostname = Environment.get("POSTGRES_HOST")
  let port =
    Environment.get("POSTGRES_PORT").flatMap(Int.init) ?? SQLPostgresConfiguration.ianaPortNumber
  let username = Environment.get("POSTGRES_USER")
  let database = Environment.get("POSTGRES_DB")

  let ssmPasswordParamName =
    Environment.get("DB_PASSWORD_SSM_PARAM_NAME") ?? "/vapor-server/database/password"
  var passwordFromSSM: String? = nil

  do {
    let region = Environment.get("AWS_REGION") ?? "us-west-2"
    let ssmClient = try SSMClient(region: region)

    app.logger.info(
      "Fetching database password from SSM Parameter Store (\(ssmPasswordParamName)) in region \(region)..."
    )

    let input = GetParameterInput(
      name: ssmPasswordParamName,
      withDecryption: true
    )
    let output = try await ssmClient.getParameter(input: input)

    if let param = output.parameter, let value = param.value {
      passwordFromSSM = value
      app.logger.info("Successfully fetched database password from SSM.")
    } else {
      app.logger.warning(
        "Password parameter '\(ssmPasswordParamName)' not found or has no value in SSM.")
    }

  } catch {
    app.logger.error(
      "Failed to fetch password from SSM: \(error). Checking environment variable as fallback.")
  }

  let finalPassword = passwordFromSSM ?? Environment.get("POSTGRES_PASSWORD")

  guard let finalUsername = username, let finalDbPassword = finalPassword,
    let finalDatabase = database,
    let finalHost = hostname
  else {
    let missingDetails = [
      hostname == nil ? "host" : nil,
      username == nil ? "user" : nil,
      finalPassword == nil ? "password (checked SSM & Env)" : nil,
      database == nil ? "database name" : nil,
    ].compactMap { $0 }.joined(separator: ", ")
    app.logger.critical(
      "Missing required database configuration details: \(missingDetails)")
    throw Abort(
      .internalServerError, reason: "Missing required database configuration: \(missingDetails)")
  }

  let finalPort = port

  let dbConfig = SQLPostgresConfiguration(
    hostname: finalHost,
    port: finalPort,
    username: finalUsername,
    password: finalDbPassword,
    database: finalDatabase,
    tls: .disable
  )
  app.databases.use(.postgres(configuration: dbConfig), as: .psql)

  // --- End Database Configuration ---

  app.migrations.add(CreateJob())
  app.migrations.add(CreateLanguages())
  app.migrations.add(CreateTechnologies())
  app.migrations.add(CreateJobLanguagePivot())
  app.migrations.add(CreateJobTechnologyPivot())

  try routes(app)

  app.middleware.use(FileMiddleware(publicDirectory: app.directory.publicDirectory))
}
