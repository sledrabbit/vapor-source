package main

import (
	"testing"
	"time"

	"gopher-source/config"
)

func TestDetermineDateRange_DefaultsToToday(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("failed to load snapshot timezone: %v", err)
	}
	frozen := time.Date(2025, time.January, 2, 8, 30, 0, 0, loc)
	withFrozenSnapshotNow(t, frozen)

	cfg := &config.Config{}

	start, end, err := determineDateRange(cfg)
	if err != nil {
		t.Fatalf("determineDateRange returned error: %v", err)
	}

	expected := frozen.Format("2006-01-02")

	if start != expected {
		t.Fatalf("expected start %s, got %s", expected, start)
	}
	if end != expected {
		t.Fatalf("expected end %s, got %s", expected, end)
	}
}

func TestDetermineDateRange_UsesProvidedDates(t *testing.T) {
	cfg := &config.Config{
		SnapshotStartDate: "2024-12-30",
		SnapshotEndDate:   "2025-01-02",
	}

	start, end, err := determineDateRange(cfg)
	if err != nil {
		t.Fatalf("determineDateRange returned error: %v", err)
	}

	if start != "2024-12-30" {
		t.Fatalf("expected start to remain %s, got %s", cfg.SnapshotStartDate, start)
	}
	if end != "2025-01-02" {
		t.Fatalf("expected end to remain %s, got %s", cfg.SnapshotEndDate, end)
	}
}

func TestDetermineDateRange_ReturnsErrorOnInvalidRange(t *testing.T) {
	cfg := &config.Config{
		SnapshotStartDate: "2025-01-02",
		SnapshotEndDate:   "2025-01-01",
	}

	_, _, err := determineDateRange(cfg)
	if err == nil {
		t.Fatal("expected error for end date before start date")
	}
}

func withFrozenSnapshotNow(t *testing.T, frozen time.Time) {
	t.Helper()

	original := snapshotNow
	snapshotNow = func() time.Time {
		return frozen
	}
	t.Cleanup(func() {
		snapshotNow = original
	})
}
