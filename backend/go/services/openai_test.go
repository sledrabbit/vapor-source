package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/openai/openai-go"
)

func TestOpenAIJobParsingSchemaAllowsNullMinYearsExperience(t *testing.T) {
	b, err := json.Marshal(OpenAIJobParsingSchema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected schema properties, got %s", b)
	}
	minYoe, ok := properties["MinYearsExperience"].(map[string]any)
	if !ok {
		t.Fatalf("expected MinYearsExperience property, got %s", b)
	}
	if _, found := minYoe["oneOf"]; found {
		t.Fatalf("OpenAI Structured Outputs does not support oneOf, got %s", b)
	}
	anyOf, ok := minYoe["anyOf"].([]any)
	if !ok || len(anyOf) != 2 {
		t.Fatalf("expected integer-or-null anyOf schema, got %s", b)
	}
	integerSchema, ok := anyOf[0].(map[string]any)
	if !ok {
		t.Fatalf("expected integer schema branch, got %s", b)
	}
	description, ok := integerSchema["description"].(string)
	if !ok || !strings.Contains(description, "Master's + 24 months OR Bachelor's + 60 months means 2") || !strings.Contains(description, "developer instruction") {
		t.Fatalf("expected schema to describe alternative paths and defer detailed rules to the developer instruction, got %s", b)
	}

	required, ok := schema["required"].([]any)
	if !ok || !containsString(required, "MinYearsExperience") {
		t.Fatalf("expected MinYearsExperience to remain required, got %s", b)
	}
}

func TestYOEInstructionsAllowSourceGroundedZero(t *testing.T) {
	instruction := jobExtractionDeveloperInstruction
	if !strings.Contains(instruction, "valid qualification path") && !strings.Contains(instruction, "valid path") {
		t.Fatal("developer instruction must evaluate valid qualification paths")
	}
	if !strings.Contains(instruction, "Return 0") || !strings.Contains(instruction, "coursework") {
		t.Fatal("developer instruction must allow source-grounded zero YOE")
	}
	if !strings.Contains(instruction, "truncated") {
		t.Fatal("developer instruction must preserve null for incomplete requirements")
	}
	if !strings.Contains(instruction, "zero-experience rule never applies to Senior") || !strings.Contains(instruction, "Never infer positive years") {
		t.Fatal("developer instruction must use seniority only to rule out zero YOE")
	}
}

func containsString(values []any, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestExecuteWithRetryDoesNotRetryCreditFailures(t *testing.T) {
	for _, tc := range []struct {
		name      string
		code      string
		errorType string
	}{
		{name: "exhausted credits", code: "credit_balance_exhausted"},
		{name: "legacy quota code", code: "insufficient_quota"},
		{name: "quota type", code: "credit_balance_exhausted", errorType: "insufficient_quota"},
		{name: "quota type without code", errorType: "insufficient_quota"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := &openai.Error{
				StatusCode: http.StatusTooManyRequests,
				Code:       tc.code,
				Type:       tc.errorType,
				Request:    &http.Request{Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: "api.openai.com", Path: "/v1/chat/completions"}},
				Response:   &http.Response{StatusCode: http.StatusTooManyRequests},
			}
			originalErr := fmt.Errorf("OpenAI API error: %w", apiErr)
			calls := 0
			_, err := (&openaiClientImpl{}).executeWithRetry(context.Background(), func() (openai.ChatCompletion, error) {
				calls++
				return openai.ChatCompletion{}, originalErr
			})
			if calls != 1 || err != originalErr || !errors.Is(err, apiErr) {
				t.Fatalf("expected one SDK operation and original error, calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestExecuteWithRetryStillRetriesTemporaryRateLimit(t *testing.T) {
	calls := 0
	result, err := (&openaiClientImpl{}).executeWithRetry(context.Background(), func() (openai.ChatCompletion, error) {
		calls++
		if calls == 1 {
			return openai.ChatCompletion{}, &openai.Error{StatusCode: http.StatusTooManyRequests, Code: "rate_limit_exceeded", Type: "rate_limit_error"}
		}
		return openai.ChatCompletion{ID: "success"}, nil
	})
	if err != nil || calls != 2 || result.ID != "success" {
		t.Fatalf("expected rate limit retried successfully, calls=%d result=%v err=%v", calls, result, err)
	}
}
