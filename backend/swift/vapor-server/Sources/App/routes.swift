import OpenAPIVapor
import Vapor

/// Registers application routes.
/// - Parameter app: The `Application` instance to register routes on.
func routes(_ app: Application) throws {
  app.get("health") { req async -> String in
    "OK"
  }

  app.get("openapi") { $0.redirect(to: "/openapi.html", redirectType: .permanent) }

  let transport = VaporTransport(routesBuilder: app)
  let handler = JobServiceAPIImpl(app: app)
  try handler.registerHandlers(on: transport, serverURL: Servers.Server1.url())
}
