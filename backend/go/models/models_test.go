package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestJobJSONIncludesZeroMinYearsExperience(t *testing.T) {
	zero := 0
	b, err := json.Marshal(Job{JobId: "entry-level", MinYearsExperience: &zero})
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}

	if !strings.Contains(string(b), `"minYearsExperience":0`) {
		t.Fatalf("expected zero min years experience in snapshot JSON, got %s", b)
	}
}

func TestJobJSONOmitsUnknownMinYearsExperience(t *testing.T) {
	b, err := json.Marshal(Job{JobId: "unknown"})
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}

	if strings.Contains(string(b), `"minYearsExperience"`) {
		t.Fatalf("expected unknown min years experience to be omitted, got %s", b)
	}
}
