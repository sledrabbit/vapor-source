package services

import (
	"encoding/json"
	"strings"
	"testing"
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
	if !ok || !strings.Contains(description, "Never return 0") || !strings.Contains(description, "Senior") || !strings.Contains(description, "positive fallback number") {
		t.Fatalf("expected schema to use seniority only to rule out zero YOE, got %s", b)
	}

	required, ok := schema["required"].([]any)
	if !ok || !containsString(required, "MinYearsExperience") {
		t.Fatalf("expected MinYearsExperience to remain required, got %s", b)
	}
}

func TestYOEInstructionsAllowSourceGroundedZero(t *testing.T) {
	for name, instruction := range map[string]string{
		"developer": jobExtractionDeveloperInstruction,
		"retry":     yoeRetryInstruction,
	} {
		if !strings.Contains(instruction, "valid qualification path") && !strings.Contains(instruction, "valid path") {
			t.Fatalf("%s instruction must evaluate valid qualification paths", name)
		}
		if !strings.Contains(instruction, "Return 0") || !strings.Contains(instruction, "coursework") {
			t.Fatalf("%s instruction must allow source-grounded zero YOE", name)
		}
		if !strings.Contains(instruction, "truncated") {
			t.Fatalf("%s instruction must preserve null for incomplete requirements", name)
		}
		if !strings.Contains(instruction, "Never return 0") || !strings.Contains(instruction, "Senior") || !strings.Contains(instruction, "positive fallback number") {
			t.Fatalf("%s instruction must use seniority only to rule out zero YOE", name)
		}
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
