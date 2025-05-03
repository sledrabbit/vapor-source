import Foundation
import Kanna
import Logging

#if canImport(FoundationNetworking)
  import FoundationNetworking
#endif

struct Scraper {
  var config: AppConfig
  let logger: Logger

  init(config: AppConfig, logger: Logger) {
    self.config = config
    self.logger = logger
  }

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
    let (data, _) = try await URLSession.shared.data(from: url)
    guard let htmlString = String(data: data, encoding: .utf8) else {
      throw URLError(.cannotDecodeContentData)
    }
    return htmlString
  }

  private func extractJobLinks(from document: HTMLDocument) throws -> [(
    url: String, jobId: String
  )] {
    let jobIdRegex = /JobID=(\d+)/
    let base = URL(string: config.scraperBaseUrl)
    var results: [(url: String, jobId: String)] = []

    for linkElement in document.css("h2.with-badge a") {
      guard let relativeUrl = linkElement["href"],
        let url = URL(string: relativeUrl, relativeTo: base)?.absoluteString,
        let match = url.firstMatch(of: jobIdRegex)
      else {
        continue
      }

      results.append((url, String(match.1)))
    }

    return results
  }

  private func parseJobDetails(from document: HTMLDocument, url: String, jobId: String) throws
    -> Job
  {
    let title = document.css("h1").first?.text ?? "Unknown Title"
    let company = document.css("h4 .capital-letter").first?.text ?? "Unknown Company"
    let location = document.css("h4 small.wrappable").first?.text ?? "Unknown Location"
    let description = document.css("span#TrackingJobBody").first?.text ?? "No description available"

    let salary =
      document.xpath(
        "//div[contains(@class,'panel-solid')]//dl//dt[contains(text(),'Salary')]/following-sibling::dd"
      ).first?.text ?? "Not specified"

    let postedDate: String
    if let dateText = document.xpath("//p[contains(text(),'Posted:')]").first?.text,
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
    query: String,
    jobIds: inout Set<String>,
    continuation: AsyncStream<Job>.Continuation
  ) async throws {
    debug("📄 Scraping page \(pageNum) of \(config.scraperMaxPages)...")

    let url = buildUrl(query: query, page: String(pageNum))
    let htmlString = try await fetchPage(url: url)
    let document = try parseHTML(from: htmlString)
    let jobLinks = try extractJobLinks(from: document)

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
            let document = try parseHTML(from: jobHtml)
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

  private func parseHTML(from htmlString: String) throws -> HTMLDocument {
    guard let document = try? HTML(html: htmlString, encoding: .utf8) else {
      throw NSError(
        domain: "ScraperError",
        code: 1,
        userInfo: [NSLocalizedDescriptionKey: "Failed to parse HTML"])
    }
    return document
  }

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
