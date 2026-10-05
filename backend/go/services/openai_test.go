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
