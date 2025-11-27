package app

import (
	"context"
	"sync"
	"testing"

	"gopher-source/config"
	"gopher-source/models"
)

type fakeParser struct {
	mu        sync.Mutex
	responses []*models.Job
	successes []bool
	callCount int
}

func (f *fakeParser) ParseWithStats(ctx context.Context, job *models.Job) (*models.Job, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.callCount >= len(f.responses) {
		return nil, false
	}
	idx := f.callCount
	f.callCount++
	return f.responses[idx], f.successes[idx]
}

type fakeDynamo struct {
	mu   sync.Mutex
	jobs []*models.Job
}

func (f *fakeDynamo) PutJob(ctx context.Context, job *models.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	copyJob := *job
	f.jobs = append(f.jobs, &copyJob)
	return nil
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
		successes: []bool{true, true},
	}
	dynamo := &fakeDynamo{}
	stats := &models.JobStats{}

	cfg := config.Config{
		MaxConcurrency: 2,
		ApiDryRun:      "false",
	}

	processAndSendJobs(context.Background(), jobsChan, stats, cfg, parser, dynamo)

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
		successes: []bool{false},
	}
	dynamo := &fakeDynamo{}
	stats := &models.JobStats{}

	cfg := config.Config{
		MaxConcurrency: 1,
		ApiDryRun:      "false",
	}

	processAndSendJobs(context.Background(), jobsChan, stats, cfg, parser, dynamo)

	snapshot := stats.Snapshot()
	if snapshot.FailedJobs != 1 {
		t.Fatalf("expected FailedJobs=1, got %d", snapshot.FailedJobs)
	}
	if len(dynamo.jobs) != 0 {
		t.Fatalf("expected no jobs persisted when parser fails, got %d", len(dynamo.jobs))
	}
}
