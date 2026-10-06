package app

import (
	"context"
	"errors"
	"sync"
	"testing"

	"gopher-source/config"
	"gopher-source/models"
)

type fakeParser struct {
	mu        sync.Mutex
	responses []*models.Job
	errors    []error
	callCount int
}

func (f *fakeParser) ParseWithStats(ctx context.Context, job *models.Job) (*models.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.callCount >= len(f.responses) {
		return nil, errors.New("missing fake response")
	}
	idx := f.callCount
	f.callCount++
	return f.responses[idx], f.errors[idx]
}

type fakeDynamo struct {
	mu     sync.Mutex
	jobs   []*models.Job
	putErr error
}

func (f *fakeDynamo) PutJob(ctx context.Context, job *models.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.putErr != nil {
		return f.putErr
	}
	copyJob := *job
	f.jobs = append(f.jobs, &copyJob)
	return nil
}

func (f *fakeDynamo) QueryJobsByPostedDate(ctx context.Context, date string) ([]models.Job, error) {
	return nil, nil
}

func (f *fakeDynamo) QueryJobsByDateRange(ctx context.Context, startDate, endDate string) ([]models.Job, error) {
	return nil, nil
}

func (f *fakeDynamo) GetAllJobIds(ctx context.Context) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func (f *fakeDynamo) WriteJobIdsToFile(filename string, keySet map[string]bool) error {
	return nil
}

func TestProcessAndSendJobsParsesAndStoresJobs(t *testing.T) {
	jobsChan := make(chan models.Job, 2)
	jobsChan <- models.Job{JobId: "1", Title: "One"}
	jobsChan <- models.Job{JobId: "2", Title: "Two"}
	close(jobsChan)

	parser := &fakeParser{
		responses: []*models.Job{
			{JobId: "1", Title: "One", IsSoftwareEngineerRelated: true},
			{JobId: "2", Title: "Two", IsSoftwareEngineerRelated: false},
		},
		errors: []error{nil, nil},
	}
	dynamo := &fakeDynamo{}
	stats := &models.JobStats{}

	cfg := config.Config{
		MaxConcurrency: 2,
		ApiDryRun:      "false",
	}

	storedIDs, err := processAndSendJobs(context.Background(), jobsChan, stats, cfg, parser, dynamo)
	if err != nil || len(storedIDs) != 2 || !storedIDs["1"] || !storedIDs["2"] {
		t.Fatalf("expected both IDs stored without error, got IDs=%v err=%v", storedIDs, err)
	}

	snapshot := stats.Snapshot()
	if snapshot.ProcessedJobs != 2 {
		t.Fatalf("expected ProcessedJobs=2, got %d", snapshot.ProcessedJobs)
	}
	if snapshot.SuccessfulJobs != 2 {
		t.Fatalf("expected SuccessfulJobs=2, got %d", snapshot.SuccessfulJobs)
	}
	if snapshot.UnrelatedJobs != 1 {
		t.Fatalf("expected UnrelatedJobs=1, got %d", snapshot.UnrelatedJobs)
	}
	if len(dynamo.jobs) != 2 {
		t.Fatalf("expected 2 jobs persisted, got %d", len(dynamo.jobs))
	}
}

func TestProcessAndSendJobsHandlesParserFailure(t *testing.T) {
	jobsChan := make(chan models.Job, 1)
	jobsChan <- models.Job{JobId: "err", Title: "Err"}
	close(jobsChan)

	parser := &fakeParser{
		responses: []*models.Job{nil},
		errors:    []error{errors.New("API unavailable")},
	}
	dynamo := &fakeDynamo{}
	stats := &models.JobStats{}

	cfg := config.Config{
		MaxConcurrency: 1,
		ApiDryRun:      "false",
	}

	storedIDs, err := processAndSendJobs(context.Background(), jobsChan, stats, cfg, parser, dynamo)
	if !errors.Is(err, parser.errors[0]) || len(storedIDs) != 0 {
		t.Fatalf("expected original parse error and no cached IDs, got IDs=%v err=%v", storedIDs, err)
	}

	snapshot := stats.Snapshot()
	if snapshot.FailedJobs != 1 {
		t.Fatalf("expected FailedJobs=1, got %d", snapshot.FailedJobs)
	}
	if len(dynamo.jobs) != 0 {
		t.Fatalf("expected no jobs persisted when parser fails, got %d", len(dynamo.jobs))
	}
}

func TestProcessAndSendJobsContinuesAfterParserFailure(t *testing.T) {
	jobsChan := make(chan models.Job, 3)
	for _, id := range []string{"first-failure", "success", "last-failure"} {
		jobsChan <- models.Job{JobId: id}
	}
	close(jobsChan)
	firstErr := errors.New("network unavailable")
	lastErr := errors.New("invalid response")
	parser := &fakeParser{
		responses: []*models.Job{nil, {JobId: "success"}, nil},
		errors:    []error{firstErr, nil, lastErr},
	}
	dynamo := &fakeDynamo{}
	stats := &models.JobStats{}
	storedIDs, err := processAndSendJobs(context.Background(), jobsChan, stats,
		config.Config{MaxConcurrency: 1}, parser, dynamo)
	if !errors.Is(err, firstErr) || !errors.Is(err, lastErr) {
		t.Fatalf("expected both original errors, got %v", err)
	}
	if parser.callCount != 3 || len(dynamo.jobs) != 1 || dynamo.jobs[0].JobId != "success" {
		t.Fatalf("expected remaining jobs processed and successful job stored, calls=%d jobs=%v", parser.callCount, dynamo.jobs)
	}
	if len(storedIDs) != 1 || !storedIDs["success"] {
		t.Fatalf("expected only successful ID eligible for cache, got %v", storedIDs)
	}
	snapshot := stats.Snapshot()
	if snapshot.ProcessedJobs != 3 || snapshot.FailedJobs != 2 || snapshot.SuccessfulJobs != 1 {
		t.Fatalf("unexpected stats: %+v", snapshot)
	}
}

func TestProcessAndSendJobsDoesNotCacheStorageFailure(t *testing.T) {
	jobsChan := make(chan models.Job, 1)
	jobsChan <- models.Job{JobId: "store-failure"}
	close(jobsChan)
	parser := &fakeParser{responses: []*models.Job{{JobId: "store-failure"}}, errors: []error{nil}}
	storeErr := errors.New("DynamoDB unavailable")
	storedIDs, err := processAndSendJobs(context.Background(), jobsChan, &models.JobStats{},
		config.Config{MaxConcurrency: 1}, parser, &fakeDynamo{putErr: storeErr})
	if !errors.Is(err, storeErr) || len(storedIDs) != 0 {
		t.Fatalf("expected storage error and no cached IDs, got IDs=%v err=%v", storedIDs, err)
	}
}

func TestProcessAndSendJobsStoresPartialParseAndReportsError(t *testing.T) {
	jobsChan := make(chan models.Job, 1)
	jobsChan <- models.Job{JobId: "partial"}
	close(jobsChan)
	retryErr := errors.New("YOE retry failed")
	parser := &fakeParser{responses: []*models.Job{{JobId: "partial"}}, errors: []error{retryErr}}
	dynamo := &fakeDynamo{}
	storedIDs, err := processAndSendJobs(context.Background(), jobsChan, &models.JobStats{},
		config.Config{MaxConcurrency: 1}, parser, dynamo)
	if !errors.Is(err, retryErr) || !storedIDs["partial"] || len(dynamo.jobs) != 1 {
		t.Fatalf("expected usable parse stored and retry error returned, IDs=%v err=%v jobs=%v", storedIDs, err, dynamo.jobs)
	}
}
