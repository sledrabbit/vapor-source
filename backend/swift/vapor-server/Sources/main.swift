import Foundation
import OpenAPIRuntime
import OpenAPIVapor
import Vapor

// struct that conforms to APIProtocol
struct JobServiceAPIImpl: APIProtocol {

    func postJobs(
        _ input: Operations.PostJobs.Input
    ) async throws -> Operations.PostJobs.Output {
        guard case .json(let job) = input.body else {
            return .badRequest(.init())
        }

        return .created(.init(body: .json(job)))
    }
}

// create Vapor app
let app: Application = try await Vapor.Application.make()

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
