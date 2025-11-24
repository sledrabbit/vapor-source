package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gopher-source/config"
	"gopher-source/models"
	"gopher-source/services"
	"gopher-source/utils"
)

// Run executes the shared scraping pipeline used by both local and Lambda binaries.
func Run(ctx context.Context, cfg *config.Config) error {
	startTime := time.Now()

	// initialize clients
	awsConfig, err := services.NewDynamoConfig(ctx)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}
	dynamoService := services.NewDynamoService(awsConfig, "Jobs")
	if err := dynamoService.CreateJobsTable(ctx); err != nil {
		return fmt.Errorf("create DynamoDB table: %w", err)
	}

	// TODO: replace local file with S3
	keySet, err := readKeysFromFile(cfg.Filename)
	if err != nil {
		return fmt.Errorf("read job ids: %w", err)
	}
	keySetInitialSize := len(keySet)

	openaiService := services.NewOpenAIService()
	parser := services.NewParserService(openaiService)
	scraper := services.NewScraperWithKeyset(*cfg, true, keySet)

	jobsChan := make(chan models.Job)
	var processingWg sync.WaitGroup
	processingWg.Add(1)
	stats := &models.JobStats{}

	go func() {
		defer processingWg.Done()
		processAndSendJobs(ctx, jobsChan, stats, *cfg, parser, dynamoService)
	}()

	// scrape
	utils.Debug(fmt.Sprintf("🚀 Starting job scraping for '%s' with max pages set to %d", cfg.DefaultQuery, cfg.MaxPages))
	scraper.ScrapeJobs(ctx, cfg.DefaultQuery, jobsChan)
	processingWg.Wait()

	// TODO: replace local file with S3
	keySet = scraper.GetProcessedIDs()
	if err := dynamoService.WriteJobIdsToFile(cfg.Filename, keySet); err != nil {
		return fmt.Errorf("write job ids: %w", err)
	}
	keySetFinalSize := len(keySet)
	utils.Debug(fmt.Sprintf("💰Jobs added to cache: %d", keySetFinalSize-keySetInitialSize))

	executionTime := time.Since(startTime)
	// print stats
	stats.PrintSummary(executionTime)

	return nil
}

func processAndSendJobs(ctx context.Context, jobsChan <-chan models.Job, stats *models.JobStats, cfg config.Config,
	parser services.ParserClient, dynamoService services.DynamoDBClient) {
	sem := make(chan struct{}, cfg.MaxConcurrency)
	var wg sync.WaitGroup

	for job := range jobsChan {
		atomic.AddInt64(&stats.TotalJobs, 1)
		wg.Add(1)
		sem <- struct{}{}

		go func(job models.Job) {
			defer wg.Done()
			defer func() { <-sem }()

			atomic.AddInt64(&stats.ProcessedJobs, 1)
			if cfg.ApiDryRun == "true" {
				mockParse(job)
				return
			}

			enhancedJob, success := parser.ParseWithStats(ctx, &job)
			if enhancedJob == nil {
				atomic.AddInt64(&stats.FailedJobs, 1)
				return
			}

			if success {
				atomic.AddInt64(&stats.SuccessfulJobs, 1)
			} else {
				atomic.AddInt64(&stats.FailedJobs, 1)
			}
			if !enhancedJob.IsSoftwareEngineerRelated {
				atomic.AddInt64(&stats.UnrelatedJobs, 1)
			}
			if err := dynamoService.PutJob(ctx, enhancedJob); err != nil {
				log.Printf("Failed to put job to DynamoDB: %v", err)
			}
			if cfg.ApiDryRun == "true" {
				mockPost(*enhancedJob)
			}
		}(job)
	}
	wg.Wait()
}

func mockPost(job models.Job) {
	jsonData, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job: %v", err)
		return
	}
	utils.Debug(fmt.Sprintf("\t📦 Post successful: %s (payload size: %d bytes)", job.Title, len(jsonData)))
}

func mockParse(job models.Job) {
	utils.Debug(fmt.Sprintf("\t🧪 DEV MODE: Simulating AI response for job: %s", job.Title))
	enhancedJob := job
	enhancedJob.ParsedDescription = fmt.Sprintf("Mock parsed description for %s", job.Title)
	enhancedJob.MinDegree = "Bachelor's"
	enhancedJob.MinYearsExperience = 3
	enhancedJob.Modality = "Remote"
	enhancedJob.Domain = "Software Development"
	enhancedJob.Languages = []string{"Go", "Python"}
	enhancedJob.Technologies = []string{"Docker", "Kubernetes"}
	mockPost(enhancedJob)
}

func readKeysFromFile(filename string) (map[string]bool, error) {
	keySet := make(map[string]bool)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			file, err = os.Create(filename)
			if err != nil {
				return keySet, fmt.Errorf("failed to create file: %v", err)
			}
			file.Close()
			return keySet, nil
		}
		return keySet, fmt.Errorf("failure to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		keySet[line] = true
	}

	if err := scanner.Err(); err != nil {
		return keySet, fmt.Errorf("error reading file: %w", err)
	}
	return keySet, nil
}
