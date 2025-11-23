import AWSDynamoDB
import Foundation

enum DynamoDBError: Error {
  /// The specified table wasn't found or couldn't be created.
  case TableNotFound
  /// The specified item wasn't found or couldn't be created.
  case ItemNotFound
  /// The Amazon DynamoDB client is not properly initialized.
  case UninitializedClient
  /// The table status reported by Amazon DynamoDB is not recognized.
  case StatusUnknown
  /// One or more specified attribute values are invalid or missing.
  case InvalidAttributes
}

public struct DynamoDBTable {
  let ddbClient: DynamoDBClient
  let tableName: String

  public init(region: String? = nil, tableName: String) async throws {
    do {
      var config = try await DynamoDBClient.DynamoDBClientConfiguration()
      if let region = region {
        config.region = region
      }

      self.ddbClient = DynamoDBClient(config: config)
      self.tableName = tableName

      try await self.createTable()
    } catch {
      print("ERROR: ", dump(error, name: "Initializing Amazon DynamoDBClient client"))
      throw error
    }
  }

  private func createTable() async throws {
    let input = CreateTableInput(
      attributeDefinitions: [
        DynamoDBClientTypes.AttributeDefinition(attributeName: "JobId", attributeType: .s),
        DynamoDBClientTypes.AttributeDefinition(attributeName: "PostedDate", attributeType: .s),
      ],
      billingMode: DynamoDBClientTypes.BillingMode.payPerRequest,
      globalSecondaryIndexes: [
        DynamoDBClientTypes.GlobalSecondaryIndex(
          indexName: "PostedDate-Index",
          keySchema: [
            DynamoDBClientTypes.KeySchemaElement(attributeName: "PostedDate", keyType: .hash),
            DynamoDBClientTypes.KeySchemaElement(attributeName: "JobId", keyType: .range),
          ],
          projection: DynamoDBClientTypes.Projection(projectionType: .all)
        )
      ],
      keySchema: [
        DynamoDBClientTypes.KeySchemaElement(attributeName: "JobId", keyType: .hash)
      ],
      tableName: self.tableName
    )
    let output = try await ddbClient.createTable(input: input)
    if output.tableDescription == nil {
      throw DynamoDBError.TableNotFound
    }
  }

  public func putJob(_ job: Job) async throws {
    do {
      let item = try await job.getAsItem()

      let input = PutItemInput(
        item: item,
        tableName: self.tableName
      )

      _ = try await self.ddbClient.putItem(input: input)
    } catch {
      print("ERROR: add job:", dump(error))
      throw error
    }
  }

  // for local testing not for AWS lambda
  public func writeJobIdsToFile(filename: String, keySet: [String: Bool]) throws {
    let cwd = FileManager.default.currentDirectoryPath
    let fileURL = URL(fileURLWithPath: cwd).appendingPathComponent(filename)

    let content = keySet.keys.joined(separator: "\n") + "\n"

    if FileManager.default.fileExists(atPath: fileURL.path) {
      if let fileHandle = try? FileHandle(forWritingTo: fileURL) {
        fileHandle.seekToEndOfFile()
        fileHandle.write(content.data(using: .utf8)!)
        fileHandle.closeFile()
      } else {
        throw NSError(
          domain: "DynamoDBError", code: -1,
          userInfo: [NSLocalizedDescriptionKey: "Could not open file for writing"])
      }
    } else {
      try content.write(to: fileURL, atomically: true, encoding: .utf8)
    }
  }
}
