import Logging
import NIOCore
import Vapor

/// The main entry point for the Vapor application.
@main
enum Entrypoint {
  /// Sets up and runs the Vapor application.
  static func main() async throws {
    var env = try Environment.detect()
    try LoggingSystem.bootstrap(from: &env)
    let app = try await Application.make(env)
    try await configure(app)
    try await app.execute()
  }
}
