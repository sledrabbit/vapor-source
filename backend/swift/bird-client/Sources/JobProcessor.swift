import Foundation
import Logging

struct JobProcessor {
  let parser: Parser
  let table: DynamoDBTable
  let config: AppConfig
  let logger: Logger

  func run(jobStream: AsyncStream<Job>) async {
    let workerLimit = max(1, config.parserMaxConcurrentTasks)
    let limiter = ConcurrencyLimiter(limit: workerLimit)

    await withTaskGroup(of: Void.self) { group in
      for await job in jobStream {
        await limiter.wait()
        group.addTask {
          defer { Task { await limiter.signal() } }
          await process(job)
        }
      }
      await group.waitForAll()
    }
  }

  private func process(_ job: Job) async {
    do {
      if let parsedJob = try await parser.parse(job: job) {
        await table.postJob(parsedJob, config: config, logger: logger)
      }
    } catch {
      logger.error("Error processing job \(job.jobId): \(error)")
    }
  }
}
