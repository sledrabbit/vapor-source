import Fluent

struct CreateJobTechnologyPivot: AsyncMigration {
  func prepare(on database: Database) async throws {
    try await database.schema("job_technology_pivot")
      .id()
      .field("job_id", .uuid, .required, .references("jobs", "id", onDelete: .cascade))
      .field(
        "technology_id", .uuid, .required, .references("technologies", "id", onDelete: .cascade)
      )
      .unique(on: "job_id", "technology_id")
      .create()
  }

  func revert(on database: Database) async throws {
    try await database.schema("job_technology_pivot").delete()
  }
}
