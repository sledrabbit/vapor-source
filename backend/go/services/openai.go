package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"

	"gopher-source/models"
)

const jobExtractionDeveloperInstruction = `You extract structured data from job postings. Read the entire title and description and prioritize extraction completeness.

For MinYearsExperience:
- Re-scan the complete source for explicit experience requirements before answering.
- When a required number or range is present, return the minimum required years (for example, 3 for "3+ years" and 2 for "2-4 years"). Do not return null in this case.
- Return 0 only when the source explicitly states that no prior professional experience is required.
- Distinguish required qualifications from experience that is only preferred or nice to have.
- Never infer a number from title or seniority labels such as Entry-level, Junior, Mid-level, Senior, Staff, Principal, Lead, Director, New Grad, or Intern.
- Return null after confirming that the source contains no explicit minimum-years requirement and does not explicitly state that no prior experience is required.

Follow the response schema exactly and do not add commentary.`

type OpenAIClient interface {
	SendMessage(ctx context.Context, message string) (openai.ChatCompletion, error)
	UnmarshalResponse(responseText string) (models.OpenAIJobParsingResponse, error)
}

type openaiClientImpl struct {
	client openai.Client
}

func NewOpenAIService() OpenAIClient {
	client := openai.NewClient()
	return &openaiClientImpl{client: client}
}

func (o *openaiClientImpl) SendMessage(ctx context.Context, message string) (openai.ChatCompletion, error) {
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "job_parsing_response",
		Description: openai.String("Parsed job posting information"),
		Schema:      OpenAIJobParsingSchema,
		Strict:      openai.Bool(true),
	}

	return o.executeWithRetry(ctx, func() (openai.ChatCompletion, error) {
		chatCompletion, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.DeveloperMessage(jobExtractionDeveloperInstruction),
				openai.UserMessage(message),
			},
			Model: openai.ChatModelGPT4_1Nano,
			ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
				OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
			},
		})
		if err != nil {
			return openai.ChatCompletion{}, fmt.Errorf("OpenAI API error: %w", err)
		}
		return *chatCompletion, nil
	})
}

func (o *openaiClientImpl) UnmarshalResponse(responseText string) (models.OpenAIJobParsingResponse, error) {
	var res models.OpenAIJobParsingResponse
	err := json.Unmarshal([]byte(responseText), &res)
	if err != nil {
		return models.OpenAIJobParsingResponse{}, fmt.Errorf("error decoding OpenAI response: %v", err)
	}
	return res, nil
}

func (o *openaiClientImpl) executeWithRetry(ctx context.Context, operation func() (openai.ChatCompletion, error)) (openai.ChatCompletion, error) {
	maxRetries := 10
	baseDelay := 1000 * time.Millisecond
	maxDelay := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		result, err := operation()
		if err != nil {
			var apiErr *openai.Error
			if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
				// exponential backoff with full jitter
				backoffDelay := time.Duration(float64(baseDelay) * math.Pow(2, float64(i)))
				backoffDelay = min(backoffDelay, maxDelay)
				delay := time.Duration(rand.Int63n(int64(backoffDelay)))
				fmt.Printf("\t⚠️ Rate limited: waiting %d ms before retry %d/%d\n", delay.Milliseconds(), i+1, maxRetries)
				time.Sleep(delay)
				continue
			}
			return openai.ChatCompletion{}, err
		}
		return result, nil
	}
	return openai.ChatCompletion{}, fmt.Errorf("\t❌ Max retry attempts of %d reached. Operation failed", maxRetries)
}

func generateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// these flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	normalizeNullableSchemas(schema)
	return schema
}

// invopop/jsonschema represents nullable fields with oneOf, while OpenAI
// Structured Outputs supports anyOf for unions. Convert nullable unions without
// changing the schemas of their integer and null branches.
func normalizeNullableSchemas(schema *jsonschema.Schema) {
	if schema == nil {
		return
	}

	if isNullableUnion(schema.OneOf) {
		schema.AnyOf = schema.OneOf
		schema.OneOf = nil
	}

	for _, child := range schema.OneOf {
		normalizeNullableSchemas(child)
	}
	for _, child := range schema.AnyOf {
		normalizeNullableSchemas(child)
	}
	if schema.Items != nil {
		normalizeNullableSchemas(schema.Items)
	}
	if schema.Properties != nil {
		for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			normalizeNullableSchemas(pair.Value)
		}
	}
	for _, definition := range schema.Definitions {
		normalizeNullableSchemas(definition)
	}
}

func isNullableUnion(schemas []*jsonschema.Schema) bool {
	for _, schema := range schemas {
		if schema != nil && schema.Type == "null" {
			return true
		}
	}
	return false
}

// generate the JSON schema at initialization time
var OpenAIJobParsingSchema = generateSchema[models.OpenAIJobParsingResponse]()
