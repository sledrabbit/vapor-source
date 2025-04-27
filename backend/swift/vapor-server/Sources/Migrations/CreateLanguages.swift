import Fluent

struct CreateLanguages: AsyncMigration {
  func prepare(on database: Database) async throws {
    try await database.schema("languages")
      .id()
      .field("name", .string, .required)
      .unique(on: "name")
      .create()
  }

  func revert(on database: Database) async throws {
    try await database.schema("languages").delete()
  }
}
