package main

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

func main() {
	startTime := time.Now()

	// load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// initialize clients
	awsConfig, err := services.NewDynamoConfig(ctx)
	if err != nil {
		fmt.Printf("failed to load aws config %v", err)
	}
	dynamoService := services.NewDynamoService(awsConfig, "Jobs")
	err = dynamoService.CreateJobsTable(ctx)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB table: %v", err)
	}
	// TODO: replace local file with S3
	keySet, err := readKeysFromFile(cfg.Filename)
	if err != nil {
		fmt.Printf("Error reading file with JobIds %v", err)
	}
	utils.Debug(fmt.Sprintf("INITIAL JobIds Length: %d", len(keySet)))
	openaiService := services.NewOpenAIService()
	parser := services.NewParserService(openaiService)
	scraper := services.NewScraperWithKeyset(*cfg, true, keySet)

	// setup concurrency
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
	dynamoService.WriteJobIdsToFile(cfg.Filename, keySet)
	utils.Debug(fmt.Sprintf("END JobIds Length: %d", len(keySet)))

	executionTime := time.Since(startTime)

	// print stats
	stats.PrintSummary(executionTime)
}

func processAndSendJobs(ctx context.Context, jobsChan <-chan models.Job, stats *models.JobStats, cfg config.Config,
	parser services.ParserClient, dynamoService services.DynamoDBClient) {
	// semaphore to limit concurrency
	sem := make(chan struct{}, cfg.MaxConcurrency)
	var wg sync.WaitGroup

	for job := range jobsChan {
		atomic.AddInt64(&stats.TotalJobs, 1)
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(job models.Job) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			atomic.AddInt64(&stats.ProcessedJobs, 1)
			if cfg.ApiDryRun == "true" {
				mockParse(job)
			} else {
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
				if enhancedJob.IsSoftwareEngineerRelated == false {
					atomic.AddInt64(&stats.UnrelatedJobs, 1)
				}
				err := dynamoService.PutJob(ctx, enhancedJob)
				if err != nil {
					log.Printf("Failed to put job to DynamoDB: %v", err)
				}
				if cfg.ApiDryRun == "true" {
					mockPost(*enhancedJob)
				}
			}
		}(job)
	}
	wg.Wait()
}

// mock send to server
func mockPost(job models.Job) {
	jsonData, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshaling job: %v", err)
		return
	}
	utils.Debug(fmt.Sprintf("\t📦 Post successful: %s (payload size: %d bytes)", job.Title, len(jsonData)))
}

// mock API parsing
func mockParse(job models.Job) {
	utils.Debug(fmt.Sprintf("\t🧪 DEV MODE: Simulating AI response for job: %s", job.Title))
	enhancedJob := job
	enhancedJob.ParsedDescription = fmt.Sprintf("Mock parsed description for %s", job.Title)
	enhancedJob.MinDegree = "Bachelor's"
	enhancedJob.MinYearsExperience = 3
	enhancedJob.Modality = "Remote"
	enhancedJob.Domain = "Software Development"
	enhancedJob.Languages = []models.Language{{Name: "Go"}, {Name: "Python"}}
	enhancedJob.Technologies = []models.Technology{{Name: "Docker"}, {Name: "Kubernetes"}}
	mockPost(enhancedJob)
}

func readKeysFromFile(filename string) (map[string]bool, error) {
	keySet := make(map[string]bool)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("failure to open file %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		keySet[line] = true
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("error reading file: %v", err)
	}
	return keySet, nil
}
