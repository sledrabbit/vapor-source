package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"gopher-source/config"
	"gopher-source/models"
)

func TestScrapeJobsEmitsUniqueJobsAndUpdatesStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "powersearch.aspx"):
			fmt.Fprintf(w, `<html><body>
				<h2 class="with-badge"><a href="http://seeker.worksourcewa.com/jobdetail.aspx?JobID=123">Job 123</a></h2>
				<h2 class="with-badge"><a href="http://seeker.worksourcewa.com/jobdetail.aspx?JobID=123">Job 123 duplicate</a></h2>
				<h2 class="with-badge"><a href="http://seeker.worksourcewa.com/jobdetail.aspx?JobID=456">Job 456</a></h2>
			</body></html>`)
		case strings.Contains(r.URL.Path, "jobdetail.aspx") && strings.Contains(r.URL.RawQuery, "JobID=123"):
			fmt.Fprintf(w, `<html><body>
				<h1 class="margin-bottom">Senior Backend Engineer</h1>
				<h4><span class="capital-letter">Acme Corp</span><small class="wrappable">Seattle, WA</small></h4>
				<span class="job-view-posting-date">2024-01-01</span>
				<p class="job-view-salary">$150k</p>
				<span id="TrackingJobBody">Build services</span>
			</body></html>`)
		case strings.Contains(r.URL.Path, "jobdetail.aspx") && strings.Contains(r.URL.RawQuery, "JobID=456"):
			fmt.Fprintf(w, `<html><body>
				<h1 class="job-view-header"></h1>
				<span class="job-view-employer"></span>
				<span class="job-view-location"></span>
				<div class="directJobBody"></div>
			</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	targetURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server url: %v", err)
	}
	origTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{
		target: targetURL,
		base:   origTransport,
	}
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	cfg := config.Config{
		BaseURL:      "http://seeker.worksourcewa.com/",
		MaxPages:     1,
		RequestDelay: 1 * time.Millisecond,
	}
	scraper := NewScraper(cfg, false)

	ctx := context.Background()
	jobsChan := make(chan models.Job)
	stats := &models.JobStats{}
	done := make(chan struct{})

	go func() {
		scraper.ScrapeJobs(ctx, "backend engineer", jobsChan, stats)
		close(done)
	}()

	var jobs []models.Job
	for job := range jobsChan {
		jobs = append(jobs, job)
	}
	<-done

	if len(jobs) != 2 {
		t.Fatalf("expected 2 unique jobs, got %d", len(jobs))
	}
	jobsByID := map[string]models.Job{}
	for _, job := range jobs {
		jobsByID[job.JobId] = job
	}
	first, ok := jobsByID["123"]
	if !ok {
		t.Fatalf("expected job 123 to be scraped, got %+v", jobsByID)
	}
	if first.Title != "Senior Backend Engineer" || first.Company != "Acme Corp" {
		t.Fatalf("unexpected parsed job fields: %+v", first)
	}
	second, ok := jobsByID["456"]
	if !ok {
		t.Fatalf("expected job 456 to be scraped, got %+v", jobsByID)
	}
	if second.Title != "Unknown Title" || second.Company != "Unknown Company" || second.Description != "No description available" {
		t.Fatalf("expected fallback values for incomplete job, got %+v", second)
	}

	snapshot := stats.Snapshot()
	if snapshot.TotalJobs != 2 {
		t.Fatalf("expected TotalJobs=2, got %d", snapshot.TotalJobs)
	}
	if snapshot.SkippedJobs != 1 {
		t.Fatalf("expected SkippedJobs=1 due to duplicate, got %d", snapshot.SkippedJobs)
	}
}

type rewriteTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "seeker.worksourcewa.com" || req.URL.Host == "worksourcewa.com" {
		req = req.Clone(req.Context())
		req.URL.Scheme = t.target.Scheme
		req.URL.Host = t.target.Host
	}
	return t.base.RoundTrip(req)
}
