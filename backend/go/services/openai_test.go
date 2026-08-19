package services

import (
	"encoding/json"
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

	required, ok := schema["required"].([]any)
	if !ok || !containsString(required, "MinYearsExperience") {
		t.Fatalf("expected MinYearsExperience to remain required, got %s", b)
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
