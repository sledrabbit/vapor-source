package main

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"gopher-source/config"
	"gopher-source/internal/app"
)

func TestErrorResponseReturnsLambdaError(t *testing.T) {
	expected := errors.New("scrape failed")
	response, err := errorResponse(http.StatusInternalServerError, expected)

	if !errors.Is(err, expected) {
		t.Fatalf("expected Lambda error %v, got %v", expected, err)
	}
	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.StatusCode)
	}
}

func TestEvaluateSnapshotTriggerInvokesForPartialResult(t *testing.T) {
	cfg := &config.Config{SnapshotLambda: "snapshot-lambda"}
	result := &app.RunResult{JobCacheEnabled: true, JobsAddedToCache: 1}
	called := false

	evaluateSnapshotTrigger(context.Background(), cfg, result, func(context.Context, *config.Config) error {
		called = true
		return nil
	})

	if !called {
		t.Fatal("expected snapshot trigger for partial result with cached jobs")
	}
}

func TestEvaluateSnapshotTriggerSkipsResultWithoutNewJobs(t *testing.T) {
	cfg := &config.Config{SnapshotLambda: "snapshot-lambda"}
	result := &app.RunResult{JobCacheEnabled: true}
	called := false

	evaluateSnapshotTrigger(context.Background(), cfg, result, func(context.Context, *config.Config) error {
		called = true
		return nil
	})

	if called {
		t.Fatal("expected snapshot trigger to be skipped without new jobs")
	}
}
