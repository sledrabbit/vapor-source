import AWSLambdaEvents
import AWSLambdaRuntime
import Foundation

@main
struct LambdaHandler: AWSLambdaRuntime.LambdaHandler {
  typealias Event = APIGatewayV2Request
  typealias Output = APIGatewayV2Response

  init(context: LambdaInitializationContext) {
    context.logger.info("Lambda Handler Initialized.")
  }

  func handle(_ event: Event, context: LambdaContext) async throws -> Output {
    let queryParams = event.queryStringParameters
    let jobQuery =
      queryParams["query"] ?? ProcessInfo.processInfo.environment["QUERY"] ?? "software engineer"
    let debugOutputString =
      queryParams["debug"] ?? ProcessInfo.processInfo.environment["DEBUG_OUTPUT"] ?? "false"
    let debugOutput = (debugOutputString.lowercased() == "true" || debugOutputString == "1")
    let apiDryRunString =
      queryParams["dryrun"] ?? ProcessInfo.processInfo.environment["API_DRY_RUN"] ?? "false"
    let apiDryRun = (apiDryRunString.lowercased() == "true" || apiDryRunString == "1")

    let config = AppConfig(
      query: jobQuery,
      promptPath: ProcessInfo.processInfo.environment["LLM_PROMPT_PATH"] ?? "",
      debugOutput: debugOutput,
      apiDryRun: apiDryRun,
      openAIApiKey: ProcessInfo.processInfo.environment["OPENAI_API_KEY"] ?? ""
    )

    context.logger.info("Starting job processing with query: \(jobQuery)")

    do {
      let runner = try AppRunner(config: config)
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
            "dryRunMode": \(apiDryRun)
          }
          """
      )
    } catch {
      context.logger.error("Error in job processing: \(error)")
      return Output(
        statusCode: .internalServerError,
        body: "Processing failed: \(error.localizedDescription)"
      )
    }
  }
}
