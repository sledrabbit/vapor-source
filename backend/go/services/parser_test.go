package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared/constant"

	"gopher-source/models"
)

type fakeOpenAIClient struct {
	sendResp         openai.ChatCompletion
	sendErr          error
	sendCalls        int
	messages         []string
	unmarshalRes     models.OpenAIJobParsingResponse
	unmarshalResults []models.OpenAIJobParsingResponse
	unmarshalErr     error
	unmarshalCalls   int
}

func (f *fakeOpenAIClient) SendMessage(ctx context.Context, message string) (openai.ChatCompletion, error) {
	f.sendCalls++
	f.messages = append(f.messages, message)
	if f.sendErr != nil {
		return openai.ChatCompletion{}, f.sendErr
	}
	return f.sendResp, nil
}

func (f *fakeOpenAIClient) UnmarshalResponse(responseText string) (models.OpenAIJobParsingResponse, error) {
	if f.unmarshalErr != nil {
		return models.OpenAIJobParsingResponse{}, f.unmarshalErr
	}
	if f.unmarshalCalls < len(f.unmarshalResults) {
		result := f.unmarshalResults[f.unmarshalCalls]
		f.unmarshalCalls++
		return result, nil
	}
	f.unmarshalCalls++
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
	if client.sendCalls != 1 {
		t.Fatalf("expected no retry for non-null YOE, got %d API calls", client.sendCalls)
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

func TestParseWithStatsRetriesNullYOE(t *testing.T) {
	fiveYears := 5
	client := &fakeOpenAIClient{
		sendResp: successfulChatCompletion(),
		unmarshalResults: []models.OpenAIJobParsingResponse{
			{ParsedDescription: "first pass", IsSoftwareEngineerRelated: true},
			{ParsedDescription: "retry", MinYearsExperience: &fiveYears, IsSoftwareEngineerRelated: true},
		},
	}
	parser := NewParserService(client)

	job, ok := parser.ParseWithStats(context.Background(), &models.Job{
		JobId:       "yoe-retry",
		Title:       "Backend Engineer",
		Description: "Required qualifications include 5+ years of software engineering experience.",
	})
	if !ok || job == nil {
		t.Fatalf("expected successful parse, got ok=%v job=%v", ok, job)
	}
	if client.sendCalls != 2 {
		t.Fatalf("expected one YOE retry, got %d API calls", client.sendCalls)
	}
	if job.MinYearsExperience == nil || *job.MinYearsExperience != 5 {
		t.Fatalf("expected retry YOE 5, got %v", job.MinYearsExperience)
	}
	if job.ParsedDescription != "first pass" {
		t.Fatalf("expected retry to update only YOE, got parsed description %q", job.ParsedDescription)
	}
	if !strings.Contains(client.messages[1], yoeRetryInstruction) {
		t.Fatalf("expected focused retry instruction, got %q", client.messages[1])
	}
}

func TestParseWithStatsRetriesNullYOEWithoutExplicitCue(t *testing.T) {
	client := &fakeOpenAIClient{
		sendResp:     successfulChatCompletion(),
		unmarshalRes: models.OpenAIJobParsingResponse{IsSoftwareEngineerRelated: true},
	}
	parser := NewParserService(client)

	job, ok := parser.ParseWithStats(context.Background(), &models.Job{
		JobId:       "yoe-retry-confirm-null",
		Title:       "Backend Engineer",
		Description: "Build and operate backend services.",
	})
	if !ok || job == nil {
		t.Fatalf("expected successful parse, got ok=%v job=%v", ok, job)
	}
	if client.sendCalls != 2 {
		t.Fatalf("expected one YOE retry, got %d API calls", client.sendCalls)
	}
	if job.MinYearsExperience != nil {
		t.Fatalf("expected confirmed unknown YOE to remain nil, got %v", *job.MinYearsExperience)
	}
	if !strings.Contains(client.messages[1], yoeRetryInstruction) {
		t.Fatalf("expected focused retry instruction, got %q", client.messages[1])
	}
}

func successfulChatCompletion() openai.ChatCompletion {
	return openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Content: "response",
					Role:    constant.Assistant("assistant"),
				},
			},
		},
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
