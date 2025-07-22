package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopher-source/config"
	"gopher-source/models"
	"gopher-source/utils"

	"github.com/gocolly/colly/v2"
)

type ScraperClient interface {
	ScrapeJobs(ctx context.Context, query string, jobsChan chan<- models.Job)
	GetProcessedIDs() map[string]bool
}

type scraperClientImpl struct {
	config       config.Config
	debugEnabled bool
	processedIDs map[string]bool
	mutex        sync.Mutex
}

func NewScraper(config config.Config, debugEnabled bool) ScraperClient {
	return &scraperClientImpl{
		config:       config,
		debugEnabled: debugEnabled,
		processedIDs: make(map[string]bool),
	}
}

func NewScraperWithKeyset(config config.Config, debugEnabled bool, existingKeySet map[string]bool) ScraperClient {
	return &scraperClientImpl{
		config:       config,
		debugEnabled: debugEnabled,
		processedIDs: existingKeySet,
	}
}

func (s *scraperClientImpl) ScrapeJobs(ctx context.Context, query string, jobsChan chan<- models.Job) {
	defer close(jobsChan)

	// create two collectors: one for search results, one for job details
	resultsCollector := colly.NewCollector(
		colly.AllowedDomains("seeker.worksourcewa.com", "worksourcewa.com"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"),
	)

	// clone collector for parallel job detail scraping
	detailsCollector := resultsCollector.Clone()
	detailsCollector.Async = true
	detailsCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 25,
	})

	currentPage := 1
	var pendingJobs sync.WaitGroup

	// extract job links from search results pages
	resultsCollector.OnHTML("h2.with-badge a", func(e *colly.HTMLElement) {
		relativeURL := e.Attr("href")
		absoluteURL := e.Request.AbsoluteURL(relativeURL)

		jobIDRegex := regexp.MustCompile(`JobID=(\d+)`)
		matches := jobIDRegex.FindStringSubmatch(absoluteURL)

		if len(matches) < 2 {
			return
		}

		jobID := matches[1]

		// prevent processing duplicate jobs
		s.mutex.Lock()
		seen := s.processedIDs[jobID]
		if !seen {
			s.processedIDs[jobID] = true
		}
		s.mutex.Unlock()

		if seen {
			utils.Debug(fmt.Sprintf("\tSkipping already processed job: %s", jobID))
			return
		}

		// queue job detail page for async processing
		pendingJobs.Add(1)
		go func(url string, jobID string) {
			defer pendingJobs.Done()

			err := detailsCollector.Visit(url)
			if err != nil {
				log.Printf("Error visiting job %s: %v", jobID, err)
			}
		}(absoluteURL, jobID)
	})

	// parse job details from individual job pages
	detailsCollector.OnHTML("body", func(e *colly.HTMLElement) {
		job := models.Job{
			URL: e.Request.URL.String(),
		}

		jobIDRegex := regexp.MustCompile(`JobID=(\d+)`)
		if matches := jobIDRegex.FindStringSubmatch(job.URL); len(matches) > 1 {
			job.JobId = matches[1]
		}

		job.Title = e.ChildText("h1.margin-bottom")
		if job.Title == "" {
			job.Title = e.ChildText("h1.job-view-header")
		}

		job.Company = e.ChildText("h4 .capital-letter")
		if job.Company == "" {
			job.Company = e.ChildText("span.job-view-employer")
		}

		job.Location = e.ChildText("h4 small.wrappable")
		if job.Location == "" {
			job.Location = e.ChildText("span.job-view-location")
		}

		dateText := e.ChildText("p:contains('Posted:')")
		if dateText != "" {
			re := regexp.MustCompile(`Posted:\s*(.+?)\s*-`)
			matches := re.FindStringSubmatch(dateText)
			if len(matches) > 1 {
				dateStr := strings.TrimSpace(matches[1])
				if parsedDate, err := time.Parse("1/2/2006", dateStr); err == nil {
					job.PostedDate = parsedDate.Format("2006-01-02")
				} else {
					job.PostedDate = dateStr
				}
			}
		}
		if job.PostedDate == "" {
			job.PostedDate = e.ChildText("span.job-view-posting-date")
		}

		// extract expiry date from HTML content
		reExpires := regexp.MustCompile(`Expires:\s*<strong>(.*?)</strong>`)
		html, err := e.DOM.Html()
		if err == nil {
			if matches := reExpires.FindStringSubmatch(html); len(matches) > 1 {
				job.ExpiresDate = strings.TrimSpace(matches[1])
			}
		}

		job.Salary = e.ChildText("p.job-view-salary")
		if job.Salary == "" {
			e.ForEach("dl span", func(_ int, el *colly.HTMLElement) {
				if strings.Contains(el.ChildText("dt"), "Salary") {
					job.Salary = el.ChildText("dd")
				}
			})
		}

		// try multiple selectors for job description
		for _, selector := range []string{
			"span#TrackingJobBody",
			"div.JobViewJobBody",
			"div.job-view-description",
			"div.directJobBody",
			"#jobViewFrame",
		} {
			job.Description = e.ChildText(selector)
			if job.Description != "" {
				break
			}
		}

		if job.Title == "" {
			job.Title = "Unknown Title"
		}
		if job.Company == "" {
			job.Company = "Unknown Company"
		}
		if job.Location == "" {
			job.Location = "Unknown Location"
		}
		if job.Description == "" {
			job.Description = "No description available"
		}
		if job.Salary == "" {
			job.Salary = "Not specified"
		}
		if job.PostedDate == "" {
			job.PostedDate = "Unknown Date"
		}

		jobsChan <- job
		utils.Debug(fmt.Sprintf("\t📋 Scraped job: %s", job.Title))
	})

	// when results page is scraped, visit next page up to max limit
	resultsCollector.OnScraped(func(r *colly.Response) {
		utils.Debug(fmt.Sprintf("✅ Completed page %d", currentPage))

		if currentPage < s.config.MaxPages {
			currentPage++
			time.Sleep(s.config.RequestDelay)

			nextPageURL := s.buildURL(query, currentPage)
			utils.Debug(fmt.Sprintf("📄 Moving to page %d: %s", currentPage, nextPageURL))

			resultsCollector.Visit(nextPageURL)
		} else {
			utils.Debug(fmt.Sprintf("⚠️ Reached maximum page limit (%d). Stopping.", s.config.MaxPages))
		}
	})

	resultsCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("❌ Error on results page: %v", err)
	})

	detailsCollector.OnError(func(r *colly.Response, err error) {
		log.Printf("❌ Error on job detail page: %v", err)
	})

	// start scraping from the first page
	startURL := s.buildURL(query, 1)
	utils.Debug(fmt.Sprintf("📄 Starting from page 1: %s", startURL))
	resultsCollector.Visit(startURL)

	pendingJobs.Wait()
	detailsCollector.Wait()

	utils.Debug("🏁 Scraping complete.")
}

func (s *scraperClientImpl) GetProcessedIDs() map[string]bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res := make(map[string]bool, len(s.processedIDs))
	for key := range s.processedIDs {
		res[key] = true
	}
	return res
}

func (s *scraperClientImpl) buildURL(query string, page int) string {
	trimQuery := strings.TrimSpace(query)
	trimQuery = strings.ReplaceAll(trimQuery, " ", "+")

	return fmt.Sprintf("%sjobsearch/powersearch.aspx?q=%s&rad_units=miles&pp=25&nosal=true&vw=b&setype=2&pg=%d&re=3",
		s.config.BaseURL, trimQuery, page)
}
