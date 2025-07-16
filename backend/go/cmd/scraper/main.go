package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
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

	// listen for SIGTERM (and SIGINT) signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	// initialize clients
	openaiService := services.NewOpenAIService()
	parser := services.NewParserService(openaiService)
	scraper := services.NewScraper(*cfg, true)

	// setup concurrency
	jobsChan := make(chan models.Job)
	var processingWg sync.WaitGroup
	processingWg.Add(1)
	stats := &models.JobStats{}

	go func() {
		defer processingWg.Done()
		processAndSendJobs(ctx, jobsChan, stats, *cfg, parser)
	}()

	// scrape
	utils.Debug(fmt.Sprintf("🚀 Starting job scraping for '%s' with max pages set to %d", cfg.DefaultQuery, cfg.MaxPages))
	scraper.ScrapeJobs(ctx, cfg.DefaultQuery, jobsChan)
	processingWg.Wait()

	executionTime := time.Since(startTime)

	// print stats
	stats.PrintSummary(executionTime)
}

func processAndSendJobs(ctx context.Context, jobsChan <-chan models.Job, stats *models.JobStats, cfg config.Config, parser services.ParserClient) {
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
				mockPost(*enhancedJob)
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
