package services

import (
	"context"
	"fmt"
	"gopher-source/models"
	"gopher-source/utils"
	"log"
	"strings"
)

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
	var message strings.Builder
	message.WriteString(job.Title)
	message.WriteString("\n")
	message.WriteString(job.Description)

	chatResp, err := p.openaiClient.SendMessage(ctx, message.String())
	if err != nil {
		log.Printf("Error sending job %s to API: %v", job.JobId, err)
		return nil, false
	}
	if len(chatResp.Choices) == 0 {
		log.Printf("no choices returned from OpenAI for job %s", job.JobId)
		return nil, false
	}
	responseText := chatResp.Choices[0].Message.Content
	res, err := p.openaiClient.UnmarshalResponse(responseText)
	if err != nil {
		log.Printf("failed to parse OpenAI response for job %s: %v", job.JobId, err)
		return nil, false
	}
	enhancedJob := *job
	populateJobFromResponse(&enhancedJob, res)
	return &enhancedJob, true
}

func populateJobFromResponse(job *models.Job, res models.OpenAIJobParsingResponse) {
	if res.IsSoftwareEngineerRelated == false {
		utils.Debug(fmt.Sprintf("\t🦉 Filtering out non-software related job (based on AI response): %s", job.Title))
	}

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

	for _, langName := range res.Languages {
		lang := models.Language{
			Name: langName,
		}
		job.Languages = append(job.Languages, lang)
	}

	for _, techName := range res.Technologies {
		tech := models.Technology{
			Name: techName,
		}
		job.Technologies = append(job.Technologies, tech)
	}

	utils.Debug(fmt.Sprintf("\t🤖 Analyzing job: %s/", job.Title))
}
