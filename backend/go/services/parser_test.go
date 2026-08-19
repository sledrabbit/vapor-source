package services

import (
	"context"
	"errors"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared/constant"

	"gopher-source/models"
)

type fakeOpenAIClient struct {
	sendResp     openai.ChatCompletion
	sendErr      error
	unmarshalRes models.OpenAIJobParsingResponse
	unmarshalErr error
}

func (f *fakeOpenAIClient) SendMessage(ctx context.Context, message string) (openai.ChatCompletion, error) {
	if f.sendErr != nil {
		return openai.ChatCompletion{}, f.sendErr
	}
	return f.sendResp, nil
}

func (f *fakeOpenAIClient) UnmarshalResponse(responseText string) (models.OpenAIJobParsingResponse, error) {
	if f.unmarshalErr != nil {
		return models.OpenAIJobParsingResponse{}, f.unmarshalErr
	}
	return f.unmarshalRes, nil
}

func TestParseWithStatsSuccess(t *testing.T) {
	minYearsExperience := 5
	client := &fakeOpenAIClient{
		sendResp: openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "response",
						Refusal: "",
						Role:    constant.Assistant("assistant"),
					},
				},
			},
		},
		unmarshalRes: models.OpenAIJobParsingResponse{
			ParsedDescription:         "parsed",
			DeadlineDate:              "tomorrow",
			MinDegree:                 "Bachelor's",
			MinYearsExperience:        &minYearsExperience,
			Modality:                  "Remote",
			Domain:                    "Backend",
			Languages:                 []string{"Go"},
			Technologies:              []string{"AWS"},
			IsSoftwareEngineerRelated: true,
		},
	}
	parser := NewParserService(client)

	job := models.Job{
		JobId:       "123",
		Title:       "Engineer",
		Description: "Desc",
	}
	enhanced, ok := parser.ParseWithStats(context.Background(), &job)
	if !ok || enhanced == nil {
		t.Fatalf("expected success, got ok=%v job=%v", ok, enhanced)
	}
	if enhanced.ParsedDescription != "parsed" || enhanced.Modality != "Remote" || enhanced.Description != "" {
		t.Fatalf("expected job fields populated, got %+v", enhanced)
	}
}

func TestPopulateJobFromResponsePreservesKnownZeroAndUnknown(t *testing.T) {
	zero := 0

	entryLevelJob := jobWithResponseMinYears(&zero)
	if entryLevelJob.MinYearsExperience == nil || *entryLevelJob.MinYearsExperience != 0 {
		t.Fatalf("expected known zero YOE to be preserved, got %v", entryLevelJob.MinYearsExperience)
	}

	unknownJob := jobWithResponseMinYears(nil)
	if unknownJob.MinYearsExperience != nil {
		t.Fatalf("expected unknown YOE to remain nil, got %v", *unknownJob.MinYearsExperience)
	}
}

func jobWithResponseMinYears(minYears *int) models.Job {
	job := models.Job{}
	populateJobFromResponse(&job, models.OpenAIJobParsingResponse{MinYearsExperience: minYears})
	return job
}

func TestParseWithStatsHandlesSendError(t *testing.T) {
	client := &fakeOpenAIClient{
		sendErr: errors.New("network"),
	}
	parser := NewParserService(client)

	if job, ok := parser.ParseWithStats(context.Background(), &models.Job{JobId: "1"}); job != nil || ok {
		t.Fatalf("expected failure when SendMessage errors, got job=%v ok=%v", job, ok)
	}
}

func TestParseWithStatsHandlesEmptyChoices(t *testing.T) {
	client := &fakeOpenAIClient{
		sendResp: openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{},
		},
	}
	parser := NewParserService(client)

	if job, ok := parser.ParseWithStats(context.Background(), &models.Job{JobId: "2"}); job != nil || ok {
		t.Fatalf("expected failure when no choices returned")
	}
}

func TestParseWithStatsHandlesUnmarshalError(t *testing.T) {
	client := &fakeOpenAIClient{
		sendResp: openai.ChatCompletion{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "invalid",
						Refusal: "",
						Role:    constant.Assistant("assistant"),
					},
				},
			},
		},
		unmarshalErr: errors.New("bad json"),
	}
	parser := NewParserService(client)

	if job, ok := parser.ParseWithStats(context.Background(), &models.Job{JobId: "3"}); job != nil || ok {
		t.Fatalf("expected failure when parsing response fails")
	}
}
