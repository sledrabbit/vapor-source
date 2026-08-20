package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopher-source/config"
	"gopher-source/models"
	"gopher-source/utils"
)

const (
	workSourceApexEndpoint = "https://worksource.my.site.com/worksourcewa/webruntime/api/apex/execute?language=en-US&asGuest=true&htmlEncode=false"
	workSourceApexClass    = "WswaJobSearchResultsController"
	defaultSearchBatchSize = 100
)

var jobHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)

type ScraperClient interface {
	ScrapeJobs(ctx context.Context, query string, jobsChan chan<- models.Job, stats *models.JobStats) error
	GetProcessedIDs() map[string]bool
}

type scraperClientImpl struct {
	config       config.Config
	debugEnabled bool
	processedIDs map[string]bool
	mutex        sync.Mutex
	httpClient   *http.Client
	apexEndpoint string
}

type scraperApexRequest struct {
	Namespace      string `json:"namespace"`
	Classname      string `json:"classname"`
	Method         string `json:"method"`
	IsContinuation bool   `json:"isContinuation"`
	Params         any    `json:"params"`
	Cacheable      bool   `json:"cacheable"`
}

type scraperApexResponse[T any] struct {
	ReturnValue T `json:"returnValue"`
}

type workSourceSearchResponse struct {
	Jobs              []workSourceJob `json:"jobs"`
	TotalCount        int             `json:"totalCount"`
	LoadMoreBatchSize int             `json:"loadMoreBatchSize"`
	Filters           json.RawMessage `json:"filters"`
	FilterMap         json.RawMessage `json:"filterMap"`
	GeoWrapper        json.RawMessage `json:"geoWrapper"`
	LastRecordID      string          `json:"lastRecordId"`
	LastPostingDate   string          `json:"lastPostingDate"`
}

type workSourceJob struct {
	RecordID     string `json:"recordId"`
	JobTitle     string `json:"jobTitle"`
	CompanyName  string `json:"companyName"`
	JobLocation  string `json:"jobLocation"`
	LocationType string `json:"locationType"`
	PostingDate  string `json:"postingDate"`
	ClosingDate  string `json:"closingDate"`
	Amount       string `json:"amount"`
	PayType      string `json:"payType"`
	Description  string `json:"description"`
}

func NewScraper(config config.Config, debugEnabled bool) ScraperClient {
	return newScraper(config, debugEnabled, make(map[string]bool))
}

func NewScraperWithKeyset(config config.Config, debugEnabled bool, existingKeySet map[string]bool) ScraperClient {
	return newScraper(config, debugEnabled, existingKeySet)
}

func newScraper(cfg config.Config, debugEnabled bool, processedIDs map[string]bool) ScraperClient {
	if processedIDs == nil {
		processedIDs = make(map[string]bool)
	}
	return &scraperClientImpl{
		config:       cfg,
		debugEnabled: debugEnabled,
		processedIDs: processedIDs,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		apexEndpoint: workSourceApexEndpoint,
	}
}

func (s *scraperClientImpl) ScrapeJobs(ctx context.Context, query string, jobsChan chan<- models.Job, stats *models.JobStats) error {
	defer close(jobsChan)

	listings, searchCalls, totalCount, searchErr := s.searchJobs(ctx, query)
	if searchErr != nil {
		log.Printf("Error searching WorkSourceWA: %v", searchErr)
		if len(listings) == 0 {
			return searchErr
		}
	}
	utils.Debug(fmt.Sprintf("WorkSourceWA returned %d of %d jobs in %d search call(s)", len(listings), totalCount, searchCalls))

	for _, listing := range listings {
		if err := ctx.Err(); err != nil {
			return err
		}
		if listing.RecordID == "" {
			log.Printf("Skipping WorkSourceWA job without a recordId")
			continue
		}

		s.mutex.Lock()
		seen := s.processedIDs[listing.RecordID]
		if !seen {
			s.processedIDs[listing.RecordID] = true
		}
		s.mutex.Unlock()

		if seen {
			utils.Debug(fmt.Sprintf("\tSkipping already processed job: %s", listing.RecordID))
			if stats != nil {
				atomic.AddInt64(&stats.SkippedJobs, 1)
			}
			continue
		}

		if stats != nil {
			atomic.AddInt64(&stats.TotalJobs, 1)
		}
		job := listing.toJob()
		select {
		case jobsChan <- job:
			utils.Debug(fmt.Sprintf("\tScraped job: %s", job.Title))
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return searchErr
}

func (s *scraperClientImpl) searchJobs(ctx context.Context, query string) ([]workSourceJob, int, int, error) {
	query = strings.TrimSpace(query)
	response, err := callScraperApex[workSourceSearchResponse](ctx, s.httpClient, s.apexEndpoint, "initializeJobSearch", map[string]any{
		"jobTitle":          query,
		"location":          nil,
		"companyId":         nil,
		"industryId":        nil,
		"currentLatitude":   nil,
		"currentLongitude":  nil,
		"additionalFilters": nil,
	})
	if err != nil {
		return nil, 1, 0, err
	}

	jobs := append([]workSourceJob(nil), response.Jobs...)
	totalCount := response.TotalCount
	batchSize := response.LoadMoreBatchSize
	if batchSize < 1 {
		batchSize = defaultSearchBatchSize
	}
	filters := response.Filters
	filterMap := response.FilterMap
	geoWrapper := response.GeoWrapper
	searchCalls := 1
	maxPages := s.config.MaxPages
	if maxPages < 1 {
		maxPages = 1
	}

	for searchCalls < maxPages && len(jobs) < totalCount {
		if len(response.Jobs) == 0 || response.LastRecordID == "" || response.LastPostingDate == "" {
			break
		}
		previousRecordID := response.LastRecordID
		previousPostingDate := response.LastPostingDate

		response, err = callScraperApex[workSourceSearchResponse](ctx, s.httpClient, s.apexEndpoint, "loadMoreJobs", map[string]any{
			"filters":         filters,
			"filterMap":       filterMap,
			"limitSize":       batchSize,
			"lastRecordId":    response.LastRecordID,
			"lastPostingDate": response.LastPostingDate,
			"geoWrapper":      geoWrapper,
		})
		searchCalls++
		if err != nil {
			return jobs, searchCalls, totalCount, err
		}
		jobs = append(jobs, response.Jobs...)
		if len(response.Jobs) == 0 || response.LastRecordID == previousRecordID && response.LastPostingDate == previousPostingDate {
			break
		}
	}

	return jobs, searchCalls, totalCount, nil
}

func callScraperApex[T any](ctx context.Context, client *http.Client, endpoint, method string, params any) (T, error) {
	var zero T
	payload, err := json.Marshal(scraperApexRequest{
		Namespace:      "",
		Classname:      workSourceApexClass,
		Method:         method,
		IsContinuation: false,
		Params:         params,
		Cacheable:      false,
	})
	if err != nil {
		return zero, fmt.Errorf("encode %s request: %w", method, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return zero, fmt.Errorf("create %s request: %w", method, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "vapor-source-scraper/2.0")

	response, err := client.Do(request)
	if err != nil {
		return zero, fmt.Errorf("send %s request: %w", method, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return zero, fmt.Errorf("read %s response: %w", method, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return zero, fmt.Errorf("%s returned HTTP %d: %s", method, response.StatusCode, truncateScraperResponse(string(body), 300))
	}

	var envelope scraperApexResponse[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, fmt.Errorf("decode %s response: %w", method, err)
	}
	return envelope.ReturnValue, nil
}

func (job workSourceJob) toJob() models.Job {
	title := fallbackScraperValue(job.JobTitle, "Unknown Title")
	company := fallbackScraperValue(job.CompanyName, "Unknown Company")
	location := fallbackScraperValue(job.JobLocation, "Unknown Location")
	description := plainTextJobDescription(job.Description)
	if description == "" {
		description = "No description available"
	}

	return models.Job{
		JobId:       job.RecordID,
		Title:       title,
		Company:     company,
		Location:    location,
		Modality:    normalizeLocationType(job.LocationType),
		PostedDate:  postingDateOnly(job.PostingDate),
		ExpiresDate: job.ClosingDate,
		PostedTime:  job.PostingDate,
		Salary:      formatWorkSourcePay(job.Amount, job.PayType),
		URL:         "https://worksource.my.site.com/worksourcewa/job-search/job-details?jobId=" + url.QueryEscape(job.RecordID),
		Description: description,
	}
}

func (s *scraperClientImpl) GetProcessedIDs() map[string]bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := make(map[string]bool, len(s.processedIDs))
	for key := range s.processedIDs {
		result[key] = true
	}
	return result
}

func postingDateOnly(value string) string {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format(time.DateOnly)
	}
	if len(value) >= len(time.DateOnly) {
		return value[:len(time.DateOnly)]
	}
	return fallbackScraperValue(value, "Unknown Date")
}

func formatWorkSourcePay(amount, payType string) string {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return "Not specified"
	}
	if !strings.HasPrefix(amount, "$") {
		amount = "$" + amount
	}
	switch strings.ToLower(strings.TrimSpace(payType)) {
	case "salary":
		return amount + "/year"
	case "hourly":
		return amount + "/hour"
	default:
		return amount
	}
}

func normalizeLocationType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "in person", "in-person", "onsite", "on-site":
		return "In-Office"
	case "hybrid":
		return "Hybrid"
	case "remote":
		return "Remote"
	default:
		return ""
	}
}

func plainTextJobDescription(value string) string {
	withoutTags := jobHTMLTagPattern.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}

func fallbackScraperValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func truncateScraperResponse(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
