import Foundation
import SwiftDotenv

func debug(_ message: String) {
  if debugOutput {
    print(message)
  }
}

do {
  try Dotenv.configure()
  debug("Dotenv configured successfully.")
} catch {
  print("🚨 Unable to configure Dotenv. Exiting. Error: \(error)")
  exit(1)
}

let debugOutputString = Dotenv["DEBUG_OUTPUT"]?.stringValue ?? ""
let debugOutput = (debugOutputString.lowercased() == "true" || debugOutputString == "1")
let apiDryRunString = Dotenv["API_DRY_RUN"]?.stringValue ?? ""
let apiDryRun = (apiDryRunString.lowercased() == "true" || apiDryRunString == "1")
let query = Dotenv["QUERY"]?.stringValue ?? "software engineer"
let promptPath = Dotenv["LLM_PROMPT_PATH"]?.stringValue ?? ""

if promptPath.isEmpty {
  print("🚨 LLM_PROMPT_PATH variable not set. Exiting.")
  exit(1)
}

debug("Using query: \(query)")
debug("Debug output enabled: \(debugOutput)")
debug("API dry run enabled: \(apiDryRun)")

let appConfig = AppConfig(
  query: query,
  promptPath: promptPath,
  debugOutput: Bool(debugOutput),
  apiDryRun: apiDryRun
)

do {
  let appRunner = try AppRunner(config: appConfig)
  await appRunner.run()
  debug("Application finished.")
} catch {
  print("🚨 Failed to initialize or run AppRunner: \(error)")
}
