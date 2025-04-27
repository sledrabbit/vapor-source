import Fluent

struct CreateTechnologies: AsyncMigration {
  func prepare(on database: Database) async throws {
    try await database.schema("technologies")
      .id()
      .field("name", .string, .required)
      .unique(on: "name")
      .create()
  }

  func revert(on database: Database) async throws {
    try await database.schema("technologies").delete()
  }
}
