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

  private func buildUrl(query: String, page: String, baseUrl: String) -> String {
    let trimQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
      .replacingOccurrences(of: " ", with: "+")
    return
      "\(baseUrl)jobsearch/powersearch.aspx?q=\(trimQuery)&rad_units=miles&pp=25&nosal=true&vw=b&setype=2&pg=\(page)&re=3"
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
    pageNum: Int, maxPages: Int, query: String, config: Config, jobIds: inout Set<String>
  ) async throws -> [Job] {
    print("Scraping page \(pageNum) of \(maxPages)...")

    let url = buildUrl(query: query, page: String(pageNum), baseUrl: config.baseUrl)
    let htmlString = try await fetchPage(url: url)
    let jobLinks = try extractJobLinks(from: htmlString, baseUrl: config.baseUrl)

    print("Found \(jobLinks.count) job links on page \(pageNum)")

    return try await processJobLinks(jobLinks: jobLinks, jobIds: &jobIds)
  }

  private func processJobLinks(jobLinks: [(url: String, jobId: String)], jobIds: inout Set<String>)
    async throws -> [Job]
  {
    try await withThrowingTaskGroup(of: Job?.self) { group -> [Job] in
      var newJobs: [Job] = []

      for (jobUrl, jobId) in jobLinks where !jobIds.contains(jobId) {
        jobIds.insert(jobId)

        group.addTask { [self] in
          do {
            print("Processing job ID: \(jobId)")
            let jobHtml = try await self.fetchPage(url: jobUrl)
            let job = try self.parseJobDetails(from: jobHtml, url: jobUrl, jobId: jobId)
            return job
          } catch {
            print("Error processing job \(jobId): \(error)")
            return nil
          }
        }
      }

      for try await job in group {
        if let job = job {
          newJobs.append(job)
          print("Completed processing job: \(job.title)")
        }
      }

      return newJobs
    }
  }

  func scrapeJobs(query: String, config: Config) async throws -> [Job] {
    var allJobs: [Job] = []
    var jobIds = Set<String>()

    print(
      "Starting job scraping with max pages set to \(config.maxPages) (maximum \(config.maxJobs) jobs)"
    )

    for pageNum in 1...config.maxPages {
      let newJobs = try await scrapeJobsFromPage(
        pageNum: pageNum,
        maxPages: config.maxPages,
        query: query,
        config: config,
        jobIds: &jobIds
      )

      if newJobs.isEmpty && pageNum > 1 {
        print("No more jobs found on page \(pageNum). Stopping.")
        break
      }

      allJobs.append(contentsOf: newJobs)
      print(
        "Completed page \(pageNum): Added \(newJobs.count) new jobs, \(allJobs.count) total jobs scraped"
      )

      if pageNum == config.maxPages {
        print("Reached maximum page limit (\(config.maxPages)). Stopping.")
      }
    }

    print("Scraping complete. Total unique jobs found: \(allJobs.count)")
    return allJobs
  }
}
