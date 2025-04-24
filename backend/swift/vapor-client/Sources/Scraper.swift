import Foundation
import SwiftSoup

struct Config {
  let maxPages: Int
  let baseUrl: String
  let requestDelay: TimeInterval

  var maxJobs: Int {
    return maxPages * 25
  }

  init(
    maxPages: Int = 2,
    baseUrl: String = "https://www.worksourcewa.com/",
    requestDelay: TimeInterval = 1.0
  ) {
    self.maxPages = maxPages
    self.baseUrl = baseUrl
    self.requestDelay = requestDelay
  }
}

struct Scraper {
  var config: Config
  let debugEnabled: Bool

  init(config: Config, debugEnabled: Bool = true) {
    self.config = config
    self.debugEnabled = debugEnabled
  }

  private func buildUrl(query: String, page: String, baseUrl: String) -> String {
    let trimQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
      .replacingOccurrences(of: " ", with: "+")
    return
      "\(baseUrl)jobsearch/PowerSearch.aspx?jt=2&q=\(trimQuery)&where=Washington&rad=30&nosal=true&sort=rv.di.dt&pp=25&rad_units=miles&vw=b&setype=2&brd=3&brd=1&pg=\(page)&re=3"
  }

  private func fetchPage(url: String) async throws -> String {
    guard let url = URL(string: url) else { throw URLError(.badURL) }
    let (data, _) = try await URLSession.shared.data(from: url)
    guard let htmlString = String(data: data, encoding: .utf8) else {
      throw URLError(.cannotDecodeContentData)
    }
    return htmlString
  }

  private func extractJobLinks(from htmlString: String, baseUrl: String) throws -> [(
    url: String, jobId: String
  )] {
    let document = try SwiftSoup.parse(htmlString)
    let jobElements = try document.select("h2.with-badge")
    let base = URL(string: baseUrl)
    let jobIdRegex = /JobID=(\d+)/

    return try jobElements.compactMap { element in
      guard let linkElement = try element.select("a").first(),
        let relativeUrl = try? linkElement.attr("href"),
        let url = URL(string: relativeUrl, relativeTo: base)?.absoluteString,
        let match = url.firstMatch(of: jobIdRegex)
      else { return nil }

      return (url, String(match.1))
    }
  }

  private func parseJobDetails(from htmlString: String, url: String, jobId: String) throws -> Job {
    let document = try SwiftSoup.parse(htmlString)

    let title = try document.select("h1").first()?.text() ?? "Unknown Title"
    let company = try document.select("h4 .capital-letter").first()?.text() ?? "Unknown Company"
    let location = try document.select("h4 small.wrappable").first()?.text() ?? "Unknown Location"
    let description =
      try document.select("span#TrackingJobBody").first()?.text() ?? "No description available"
    let salary =
      try document.select("div.panel-solid dl span:has(dt:contains(Salary)) dd").first()?.text()
      ?? "Not specified"

    let postedDate: String
    if let dateText = try document.select("p:contains(Posted:)").first()?.text(),
      let match = dateText.firstMatch(of: /Posted:\s*(.+?)(?:\s*-|$)/)
    {
      postedDate = String(match.1).trimmingCharacters(in: .whitespacesAndNewlines)
    } else {
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

  private func scrapeJobsFromPage(
    pageNum: Int,
    maxPages: Int,
    query: String,
    config: Config,
    jobIds: inout Set<String>,
    continuation: AsyncStream<Job>.Continuation
  ) async throws {
    debug("📄 Scraping page \(pageNum) of \(maxPages)...")

    let url = buildUrl(query: query, page: String(pageNum), baseUrl: config.baseUrl)
    let htmlString = try await fetchPage(url: url)
    let jobLinks = try extractJobLinks(from: htmlString, baseUrl: config.baseUrl)

    debug("🔍 Found \(jobLinks.count) job links on page \(pageNum)")

    await processJobLinks(jobLinks: jobLinks, jobIds: &jobIds, continuation: continuation)
  }

  private func processJobLinks(
    jobLinks: [(url: String, jobId: String)],
    jobIds: inout Set<String>,
    continuation: AsyncStream<Job>.Continuation
  ) async {
    await withTaskGroup(of: Job?.self) { group -> Void in
      for (jobUrl, jobId) in jobLinks where !jobIds.contains(jobId) {
        jobIds.insert(jobId)

        group.addTask { [self] in
          do {
            let jobHtml = try await self.fetchPage(url: jobUrl)
            let job = try self.parseJobDetails(from: jobHtml, url: jobUrl, jobId: jobId)
            return job
          } catch {
            print("Error processing job \(jobId): \(error)")
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

  func scrapeJobs(query: String, config: Config) -> AsyncStream<Job> {
    return AsyncStream { continuation in
      Task {
        do {
          var jobIds = Set<String>()

          debug(
            "🚀 Starting job scraping with max pages set to \(config.maxPages) (maximum \(config.maxJobs) jobs)"
          )

          for pageNum in 1...config.maxPages {
            do {
              try await scrapeJobsFromPage(
                pageNum: pageNum,
                maxPages: config.maxPages,
                query: query,
                config: config,
                jobIds: &jobIds,
                continuation: continuation
              )
              debug("✅ Completed page \(pageNum)")
            } catch {
              print("❌ Error scraping page \(pageNum): \(error). Continuing to next page.")
            }

            if pageNum == config.maxPages {
              print("⚠️ Reached maximum page limit (\(config.maxPages)). Stopping.")
            }
          }
          print("🏁 Scraping complete.")
          continuation.finish()
        }
      }
    }
  }
}
