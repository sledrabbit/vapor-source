import Foundation
import Logging
import SwiftSoup

#if canImport(FoundationNetworking)
  import FoundationNetworking
#endif

// MARK: - Dependencies

protocol NetworkFetching: Sendable {
  func fetchData(from url: URL) async throws -> Data
}

protocol HtmlParsing: Sendable {
  func parseHtml(from htmlString: String) throws -> Document
}

extension URLSession: NetworkFetching {
  func fetchData(from url: URL) async throws -> Data {
    let (data, _) = try await data(from: url)
    return data
  }
}

struct SwiftSoupHtmlParser: HtmlParsing {
  public init() {}

  public func parseHtml(from htmlString: String) throws -> Document {
    return try SwiftSoup.parse(htmlString)
  }
}

struct Scraper: Sendable {
  let config: AppConfig
  let logger: Logger
  let networkFetcher: NetworkFetching
  let htmlParser: HtmlParsing

  init(
    config: AppConfig,
    logger: Logger,
    networkFetcher: NetworkFetching? = nil,
    htmlParser: HtmlParsing? = nil
  ) {
    self.config = config
    self.logger = logger
    self.networkFetcher = networkFetcher ?? URLSession.shared
    self.htmlParser = htmlParser ?? SwiftSoupHtmlParser()
  }
}

extension Scraper {
  func scrapeJobs(query: String) -> AsyncStream<Job> {
    return AsyncStream { continuation in
      Task {
        do {
          var jobIds = Set<String>()

          debug(
            "🚀 Starting job scraping with max pages set to \(config.scraperMaxPages) (maximum \(config.scraperMaxPages) jobs)"
          )

          for pageNum in 1...config.scraperMaxPages {
            do {
              try await scrapeJobsFromPage(
                pageNum: pageNum,
                query: query,
                jobIds: &jobIds,
                continuation: continuation
              )
              debug("✅ Completed page \(pageNum)")
            } catch {
              logger.error("❌ Error scraping page \(pageNum): \(error). Continuing to next page.")
            }

            if pageNum == config.scraperMaxPages {
              logger.warning("⚠️ Reached maximum page limit (\(config.scraperMaxPages)). Stopping.")
            }
          }
          logger.info("🏁 Scraping complete.")
          continuation.finish()
        }
      }
    }
  }
}

// MARK: - Private Functions

extension Scraper {
  private func debug(_ message: String) {
    if config.debugOutput {
      logger.info("\(message)")
    }
  }

  private func buildUrl(query: String, page: String) -> String {
    let trimQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
      .replacingOccurrences(of: " ", with: "+")
    return
      "\(config.scraperBaseUrl)jobsearch/PowerSearch.aspx?jt=2&q=\(trimQuery)&where=Washington&rad=30&nosal=true&sort=rv.di.dt&pp=25&rad_units=miles&vw=b&setype=2&brd=3&brd=1&pg=\(page)&re=3"
  }

  private func fetchPage(url: String) async throws -> String {
    guard let url = URL(string: url) else { throw URLError(.badURL) }
    let data = try await networkFetcher.fetchData(from: url)
    guard let htmlString = String(data: data, encoding: .utf8) else {
      throw URLError(.cannotDecodeContentData)
    }
    return htmlString
  }

  private func extractJobLinks(from document: Document) throws -> [(
    url: String, jobId: String
  )] {
    let jobIdRegex = /JobID=(\d+)/
    let base = URL(string: config.scraperBaseUrl)
    var results: [(url: String, jobId: String)] = []

    let linkElements = try document.select("h2.with-badge a")
    for linkElement in linkElements {
      guard let relativeUrl = try? linkElement.attr("href"),
        let url = URL(string: relativeUrl, relativeTo: base)?.absoluteString,
        let match = url.firstMatch(of: jobIdRegex)
      else {
        continue
      }

      results.append((url, String(match.1)))
    }

    return results
  }

  private func parseJobDetails(from document: Document, url: String, jobId: String) throws -> Job {
    let title = try document.select("h1").first()?.text() ?? "Unknown Title"
    let company = try document.select("h4 .capital-letter").first()?.text() ?? "Unknown Company"
    let location = try document.select("h4 small.wrappable").first()?.text() ?? "Unknown Location"
    let description =
      try document.select("span#TrackingJobBody").first()?.text() ?? "No description available"

    let salary =
      try document.select("div.panel-solid dl span:has(dt:contains(Salary)) dd").first()?.text()
      ?? "Not specified"

    var postedDate = ""
    if let dateText = try document.select("p:contains(Posted:)").first()?.text(),
      let match = dateText.firstMatch(of: /Posted:\s*(.+?)(?:\s*-|$)/)
    {
      let rawDate = String(match.1).trimmingCharacters(in: .whitespacesAndNewlines)
      postedDate = normalizePostedDate(rawDate)
    }

    if postedDate.isEmpty,
      let fallback = try document.select("span.job-view-posting-date").first()?.text(),
      !fallback.isEmpty
    {
      postedDate = fallback
    }

    if postedDate.isEmpty {
      postedDate = "Unknown Date"
    }

    return Job(
      id: nil,
      jobId: jobId,
      title: title,
      company: company,
      location: location,
      modality: nil,
      postedDate: postedDate,
      expiresDate: nil,
      salary: salary,
      url: url,
      minYearsExperience: nil,
      minDegree: nil,
      domain: nil,
      description: description,
      parsedDescription: nil,
      s3Pointer: nil,
      languages: nil,
      technologies: nil
    )
  }

  private func normalizePostedDate(_ dateString: String) -> String {
    let inputFormatter = DateFormatter()
    inputFormatter.locale = Locale(identifier: "en_US_POSIX")
    inputFormatter.timeZone = TimeZone(secondsFromGMT: 0)
    inputFormatter.dateFormat = "M/d/yyyy"

    guard let date = inputFormatter.date(from: dateString) else {
      return dateString
    }

    let outputFormatter = DateFormatter()
    outputFormatter.locale = Locale(identifier: "en_US_POSIX")
    outputFormatter.timeZone = TimeZone(secondsFromGMT: 0)
    outputFormatter.dateFormat = "yyyy-MM-dd"

    return outputFormatter.string(from: date)
  }

  private func scrapeJobsFromPage(
    pageNum: Int,
    query: String,
    jobIds: inout Set<String>,
    continuation: AsyncStream<Job>.Continuation
  ) async throws {
    debug("📄 Scraping page \(pageNum) of \(config.scraperMaxPages)...")

    let url = buildUrl(query: query, page: String(pageNum))
    let htmlString = try await fetchPage(url: url)
    let document = try htmlParser.parseHtml(from: htmlString)
    let jobLinks = try extractJobLinks(from: document)

    debug("🔍 Found \(jobLinks.count) job links on page \(pageNum)")

    await processJobLinks(jobLinks: jobLinks, jobIds: &jobIds, continuation: continuation)
  }

  private func processJobLinks(
    jobLinks: [(url: String, jobId: String)],
    jobIds: inout Set<String>,
    continuation: AsyncStream<Job>.Continuation
  ) async {
    let fetchLimit = max(1, config.scraperMaxConcurrentRequests)
    let limiter = ConcurrencyLimiter(limit: fetchLimit)
    await withTaskGroup(of: Job?.self) { group -> Void in
      for (jobUrl, jobId) in jobLinks where !jobIds.contains(jobId) {
        await limiter.wait()
        jobIds.insert(jobId)

        group.addTask { [self] in
          defer { Task { await limiter.signal() } }
          do {
            let jobHtml = try await self.fetchPage(url: jobUrl)
            let document = try htmlParser.parseHtml(from: jobHtml)
            let job = try self.parseJobDetails(from: document, url: jobUrl, jobId: jobId)
            return job
          } catch {
            logger.error("Error processing job \(jobId): \(error)")
            return nil
          }
        }
      }

      for await job in group {
        if let job = job {
          continuation.yield(job)
          debug("\t📋 Scraped job: \(job.title)")
        }
      }
    }
  }
}
