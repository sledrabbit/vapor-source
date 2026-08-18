package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"gopher-source/config"
	"gopher-source/models"
)

func TestScrapeJobsPaginatesEmitsUniqueJobsAndUpdatesStats(t *testing.T) {
	var callsMu sync.Mutex
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var request scraperApexRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		callsMu.Lock()
		methods = append(methods, request.Method)
		callsMu.Unlock()

		switch request.Method {
		case "initializeJobSearch":
			writeSearchResponse(t, w, workSourceSearchResponse{
				Jobs: []workSourceJob{
					{
						RecordID:     "record-123",
						JobTitle:     "Senior Backend Engineer",
						CompanyName:  "Acme Corp",
						JobLocation:  "Seattle, WA",
						LocationType: "Hybrid",
						PostingDate:  "2026-08-18T01:40:56.000Z",
						ClosingDate:  "2026-09-18",
						Amount:       "150,000 - $180,000",
						PayType:      "Salary",
						Description:  "<p>Build &amp; maintain services.</p>",
					},
					{RecordID: "record-123", JobTitle: "Duplicate"},
				},
				TotalCount:        3,
				LoadMoreBatchSize: 100,
				Filters:           json.RawMessage(`"active jobs"`),
				FilterMap:         json.RawMessage(`{"limitSize":100}`),
				GeoWrapper:        json.RawMessage(`null`),
				LastRecordID:      "record-123",
				LastPostingDate:   "2026-08-18T01:40:56.000Z",
			})
		case "loadMoreJobs":
			var params struct {
				LastRecordID    string `json:"lastRecordId"`
				LastPostingDate string `json:"lastPostingDate"`
				LimitSize       int    `json:"limitSize"`
			}
			encoded, err := json.Marshal(request.Params)
			if err != nil {
				t.Errorf("encode params: %v", err)
			}
			if err := json.Unmarshal(encoded, &params); err != nil {
				t.Errorf("decode params: %v", err)
			}
			if params.LastRecordID != "record-123" || params.LastPostingDate == "" || params.LimitSize != 100 {
				t.Errorf("unexpected loadMoreJobs params: %+v", params)
			}
			writeSearchResponse(t, w, workSourceSearchResponse{
				Jobs:            []workSourceJob{{RecordID: "record-456"}},
				LastRecordID:    "record-456",
				LastPostingDate: "2026-08-17T00:00:00.000Z",
			})
		default:
			http.Error(w, "unexpected method", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfg := config.Config{MaxPages: 2}
	scraper := NewScraper(cfg, false)
	impl := scraper.(*scraperClientImpl)
	impl.httpClient = server.Client()
	impl.apexEndpoint = server.URL

	jobsChan := make(chan models.Job)
	stats := &models.JobStats{}
	go scraper.ScrapeJobs(context.Background(), "backend engineer", jobsChan, stats)

	var jobs []models.Job
	for job := range jobsChan {
		jobs = append(jobs, job)
	}

	if fmt.Sprint(methods) != "[initializeJobSearch loadMoreJobs]" {
		t.Fatalf("unexpected Apex calls: %v", methods)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 unique jobs, got %d", len(jobs))
	}
	jobsByID := map[string]models.Job{}
	for _, job := range jobs {
		jobsByID[job.JobId] = job
	}

	first, ok := jobsByID["record-123"]
	if !ok {
		t.Fatalf("expected record-123, got %+v", jobsByID)
	}
	if first.Title != "Senior Backend Engineer" || first.Company != "Acme Corp" || first.Location != "Seattle, WA" {
		t.Fatalf("unexpected mapped fields: %+v", first)
	}
	if first.Description != "Build & maintain services." {
		t.Fatalf("unexpected description: %q", first.Description)
	}
	if first.PostedDate != "2026-08-18" || first.PostedTime != "2026-08-18T01:40:56.000Z" {
		t.Fatalf("unexpected posting dates: %+v", first)
	}
	if first.ExpiresDate != "2026-09-18" || first.Salary != "$150,000 - $180,000/year" || first.Modality != "Hybrid" {
		t.Fatalf("unexpected normalized fields: %+v", first)
	}
	if first.URL != "https://worksource.my.site.com/worksourcewa/job-search/job-details?jobId=record-123" {
		t.Fatalf("unexpected listing URL: %q", first.URL)
	}

	second, ok := jobsByID["record-456"]
	if !ok {
		t.Fatalf("expected record-456, got %+v", jobsByID)
	}
	if second.Title != "Unknown Title" || second.Company != "Unknown Company" || second.Description != "No description available" {
		t.Fatalf("expected fallback values, got %+v", second)
	}

	snapshot := stats.Snapshot()
	if snapshot.TotalJobs != 2 {
		t.Fatalf("expected TotalJobs=2, got %d", snapshot.TotalJobs)
	}
	if snapshot.SkippedJobs != 1 {
		t.Fatalf("expected SkippedJobs=1, got %d", snapshot.SkippedJobs)
	}
}

func writeSearchResponse(t *testing.T, w http.ResponseWriter, response workSourceSearchResponse) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scraperApexResponse[workSourceSearchResponse]{ReturnValue: response}); err != nil {
		t.Errorf("encode response: %v", err)
	}
}
