package services

import (
	"context"
	"fmt"
	"gopher-source/models"
	"gopher-source/utils"
	"log"
	"regexp"
	"strings"
	"time"
)

var (
	yearsMentionPattern = regexp.MustCompile(`(?i)\b(?:\d{1,2}|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)(?:\s*(?:\+|plus)|\s*(?:[-–—]|to)\s*(?:\d{1,2}|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty))?\s*(?:years?|yrs?)\b`)
	noExperiencePattern = regexp.MustCompile(`(?i)\b(?:(?:no|zero)\s+(?:(?:prior|previous|professional|industry|work)\s+)*experience\s+(?:is\s+)?(?:required|necessary|needed)|experience\s+is\s+not\s+(?:required|necessary|needed))\b`)
)

const yoeRetryInstruction = `The previous extraction returned null for MinYearsExperience, but the source contains possible explicit experience evidence. Re-scan the entire source and distinguish required from preferred qualifications. Return the lower bound of an explicit required number or range, or 0 only when the source explicitly says no prior professional experience is required. Do not infer a number from the title or seniority. Return null only if the detected phrase does not establish a minimum requirement. Return the complete structured response.`

type ParserClient interface {
	ParseWithStats(ctx context.Context, job *models.Job) (*models.Job, bool)
}

type parserClientImpl struct {
	openaiClient OpenAIClient
}

func NewParserService(openaiClient OpenAIClient) ParserClient {
	return &parserClientImpl{openaiClient: openaiClient}
}

func (p *parserClientImpl) ParseWithStats(ctx context.Context, job *models.Job) (*models.Job, bool) {
	message := buildJobParsingMessage(job)
	res, err := p.parseMessage(ctx, message)
	if err != nil {
		log.Printf("Error sending job %s to API: %v", job.JobId, err)
		return nil, false
	}

	if res.MinYearsExperience == nil && shouldRetryYOE(job) {
		retryRes, retryErr := p.parseMessage(ctx, yoeRetryInstruction+"\n\n"+message)
		if retryErr != nil {
			log.Printf("YOE retry failed for job %s: %v", job.JobId, retryErr)
		} else if retryRes.MinYearsExperience != nil {
			res.MinYearsExperience = retryRes.MinYearsExperience
		}
	}

	enhancedJob := *job
	populateJobFromResponse(&enhancedJob, res)
	return &enhancedJob, true
}

func (p *parserClientImpl) parseMessage(ctx context.Context, message string) (models.OpenAIJobParsingResponse, error) {
	chatResp, err := p.openaiClient.SendMessage(ctx, message)
	if err != nil {
		return models.OpenAIJobParsingResponse{}, err
	}
	if len(chatResp.Choices) == 0 {
		return models.OpenAIJobParsingResponse{}, fmt.Errorf("no choices returned from OpenAI")
	}

	responseText := chatResp.Choices[0].Message.Content
	res, err := p.openaiClient.UnmarshalResponse(responseText)
	if err != nil {
		return models.OpenAIJobParsingResponse{}, fmt.Errorf("parse OpenAI response: %w", err)
	}
	return res, nil
}

func buildJobParsingMessage(job *models.Job) string {
	var message strings.Builder
	message.WriteString("Job title: ")
	message.WriteString(job.Title)
	message.WriteString("\n\nJob description:\n")
	message.WriteString(job.Description)
	return message.String()
}

func containsYOECue(source string) bool {
	// Favor a single extra adjudication call over missing a less conventionally
	// worded requirement. The retry prompt tells the model to keep null when the
	// years mention is unrelated to job experience.
	return yearsMentionPattern.MatchString(source)
}

func shouldRetryYOE(job *models.Job) bool {
	source := job.Title + "\n" + job.Description
	return containsYOECue(source) || noExperiencePattern.MatchString(source)
}

func populateJobFromResponse(job *models.Job, res models.OpenAIJobParsingResponse) {
	if !res.IsSoftwareEngineerRelated {
		utils.Debug(fmt.Sprintf("\t🦉 Filtering out non-software related job (based on AI response): %s", job.Title))
	}

	job.Description = ""
	job.ParsedDescription = res.ParsedDescription
	job.ExpiresDate = res.DeadlineDate
	job.MinDegree = res.MinDegree
	job.MinYearsExperience = res.MinYearsExperience
	job.IsSoftwareEngineerRelated = res.IsSoftwareEngineerRelated

	if res.Modality != "" {
		job.Modality = res.Modality
	}

	if res.Domain != "" {
		job.Domain = res.Domain
	}

	job.Languages = res.Languages
	job.Technologies = res.Technologies
	job.PostedTime = time.Now().UTC().Format(time.RFC3339Nano)

	utils.Debug(fmt.Sprintf("\t🤖 Analyzing job: %s/", job.Title))
}
