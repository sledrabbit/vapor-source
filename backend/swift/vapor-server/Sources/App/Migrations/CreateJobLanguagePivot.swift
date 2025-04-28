import Fluent

struct CreateJobLanguagePivot: AsyncMigration {
  func prepare(on database: Database) async throws {
    try await database.schema("job_language_pivot")
      .id()
      .field("job_id", .uuid, .required, .references("jobs", "id", onDelete: .cascade))
      .field("language_id", .uuid, .required, .references("languages", "id", onDelete: .cascade))
      .unique(on: "job_id", "language_id")
      .create()
  }

  func revert(on database: Database) async throws {
    try await database.schema("job_language_pivot").delete()
  }
}
