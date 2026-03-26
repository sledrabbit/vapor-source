import Foundation
import Logging

struct JobProcessor {
  let parser: Parser
  let table: DynamoDBTable
  let config: AppConfig
  let logger: Logger

  func run(jobStream: AsyncStream<Job>) async {
    let workerLimit = max(1, config.parserMaxConcurrentTasks)

    await withTaskGroup(of: Void.self) { group in
      var inFlight = 0

      for await job in jobStream {
        if inFlight >= workerLimit {
          await group.next()
          inFlight -= 1
        }

        inFlight += 1
        group.addTask {
          await process(job)
        }
      }

      while inFlight > 0 {
        await group.next()
        inFlight -= 1
      }
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
