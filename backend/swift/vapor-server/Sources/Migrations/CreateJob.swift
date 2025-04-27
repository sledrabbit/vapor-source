import Fluent

struct CreateJob: Migration {
  // Creates the schema
  func prepare(on database: Database) -> EventLoopFuture<Void> {
    database.schema(Job.schema)  // Use the schema name from the Model
      .id()
      .field("job_id", .string, .required)
      .field("title", .string, .required)
      .field("company", .string, .required)
      .field("location", .string, .required)
      .field("posted_date", .string, .required)
      .field("salary", .string, .required)
      .field("url", .string, .required)
      .field("description", .string, .required)
      .field("modality", .string)
      .field("expires_date", .string)
      .field("min_years_experience", .int)
      .field("min_degree", .string)
      .field("domain", .string)
      .field("parsed_description", .string)
      .field("s3_pointer", .string)
      .unique(on: "job_id")
      .create()
  }

  // Reverts the schema creation (used for rollbacks)
  func revert(on database: Database) -> EventLoopFuture<Void> {
    database.schema(Job.schema).delete()
  }
}
