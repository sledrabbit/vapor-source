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
		return models.OpenAIJobParsingResponse{}, fmt.Errorf("Error decoding OpenAI response: %v", err)
	}
	return res, nil
}

func (o *openaiClientImpl) executeWithRetry(ctx context.Context, operation func() (openai.ChatCompletion, error)) (openai.ChatCompletion, error) {
	maxRetries := 10
	baseDelay := 500 * time.Millisecond
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
	return openai.ChatCompletion{}, fmt.Errorf("max retries (%d) reached for rate limiting", maxRetries)
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
	return schema
}

// generate the JSON schema at initialization time
var OpenAIJobParsingSchema = generateSchema[models.OpenAIJobParsingResponse]()
