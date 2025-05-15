import Foundation
import Logging

actor ConcurrencyLimiter {
  private let limit: Int
  private var currentCount = 0
  private var waitQueue: [CheckedContinuation<Void, Never>] = []

  init(limit: Int) {
    self.limit = limit
  }

  func wait() async {
    await withCheckedContinuation { continuation in
      if currentCount < limit {
        currentCount += 1
        continuation.resume()
      } else {
        waitQueue.append(continuation)
      }
    }
  }

  func signal() async {
    if let continuation = waitQueue.first {
      waitQueue.removeFirst()
      continuation.resume()
    } else {
      currentCount -= 1
    }
  }
}

func retryWithBackoff<T>(
  maxAttempts: Int = 10,
  initialDelay: TimeInterval = 1.0,
  backoffFactor: Double = 2.0,
  jitterFactor: Double = 0.1,
  logger: Logger,
  operation: @escaping () async throws -> T
) async throws -> T {
  var attempts = 0
  var currentDelay = initialDelay

  while attempts < maxAttempts {
    attempts += 1
    do {
      return try await operation()
    } catch {
      if attempts == maxAttempts {
        logger.error(
          "❌ Max retry attempts (\(maxAttempts)) reached. Operation failed. Error: \(error)")
        throw error
      }
      let jitter = Double.random(in: -jitterFactor...jitterFactor) * currentDelay
      let delayWithJitter = max(0, currentDelay + jitter)
      let delayInSeconds = String(format: "%.2f", delayWithJitter)
      logger.warning(
        "⚠️ Attempt \(attempts)/\(maxAttempts) failed. Retrying in \(delayInSeconds)s... Error: \(error)"
      )
      try await Task.sleep(nanoseconds: UInt64(delayWithJitter * 1_000_000_000))
      currentDelay *= backoffFactor
    }
  }
  fatalError("Retry logic exited loop unexpectedly.")
}
