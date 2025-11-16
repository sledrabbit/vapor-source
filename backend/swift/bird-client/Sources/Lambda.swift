import AWSLambdaEvents
import AWSLambdaRuntime
import Foundation

struct MyLambdaHandler: LambdaHandler {
  typealias Event = APIGatewayV2Request
  typealias Output = APIGatewayV2Response

  func handle(_ event: Event, context: LambdaContext) async throws
    -> Output
  {
    let config = try AppConfig()
    context.logger.info("Configuration loaded successfully.")
    context.logger.info("Lambda Handler Initialized.")

    let queryParams = event.queryStringParameters
    let jobQuery =
      queryParams["query"] ?? ProcessInfo.processInfo.environment["QUERY"] ?? "software engineer"
    let debugOutputString =
      queryParams["debug"] ?? ProcessInfo.processInfo.environment["DEBUG_OUTPUT"] ?? "false"
    let debugOutput = (debugOutputString.lowercased() == "true" || debugOutputString == "1")
    let apiDryRunString =
      queryParams["dryrun"] ?? ProcessInfo.processInfo.environment["API_DRY_RUN"] ?? "false"
    let apiDryRun = (apiDryRunString.lowercased() == "true" || apiDryRunString == "1")
    let useMockJobsString =
      queryParams["mock"] ?? ProcessInfo.processInfo.environment["USE_MOCK_JOBS"] ?? "false"
    let useMockJobs = (useMockJobsString.lowercased() == "true" || useMockJobsString == "1")

    context.logger.info("Starting job processing with query: \(jobQuery)")

    let runner = try AppRunner(config: config, logger: context.logger)
    await runner.run()
    let result = "Job processing completed successfully"
    context.logger.info("\(result)")

    return Output(
      statusCode: .ok,
      body: """
        {
          "message": "Job processing completed",
          "query": "\(jobQuery)",
          "debugMode": \(debugOutput),
          "dryRunMode": \(apiDryRun),
          "useMockJobs": \(useMockJobs)
        }
        """
    )
  }
}
