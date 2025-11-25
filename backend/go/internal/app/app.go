package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type RunResult struct {
	ExecutionTime       time.Duration
	Stats               models.JobStats
	JobCacheEnabled     bool
	JobCacheInitialSize int
	JobCacheFinalSize   int
	JobsAddedToCache    int
	JobCacheS3Bucket    string
	JobCacheS3Key       string
}

// Run executes the shared scraping pipeline used by both local and Lambda binaries.
func Run(ctx context.Context, cfg *config.Config) (*RunResult, error) {
	startTime := time.Now()
	result := &RunResult{
		JobCacheEnabled: cfg.UseJobIDFile,
	}

	// initialize clients
	awsConfig, err := services.NewDynamoConfig(ctx, cfg.AWSRegion)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	dynamoService := services.NewDynamoService(awsConfig, cfg.DynamoTableName, cfg.DynamoEndpoint)

	var s3Service services.S3Client
	if cfg.UseJobIDFile && cfg.UseS3JobIDFile {
		if cfg.JobIDsBucket != "" && cfg.JobIDsS3Key != "" {
			s3Service = services.NewS3Service(awsConfig)
			utils.Debug(fmt.Sprintf("Downloading job ID cache from s3://%s/%s", cfg.JobIDsBucket, cfg.JobIDsS3Key))
			if err := downloadJobIDCache(ctx, s3Service, cfg.JobIDsBucket, cfg.JobIDsS3Key, cfg.Filename); err != nil {
				return nil, fmt.Errorf("sync job id cache: %w", err)
			}
		} else {
			utils.Debug("S3 job ID cache enabled but JOB_IDS_BUCKET or JOB_IDS_S3_KEY not set; skipping S3 sync")
		}
	}

	keySet := make(map[string]bool)
	keySetInitialSize := 0
	if cfg.UseJobIDFile {
		var err error
		keySet, err = readKeysFromFile(cfg.Filename)
		if err != nil {
			return nil, fmt.Errorf("read job ids: %w", err)
		}
		keySetInitialSize = len(keySet)
		result.JobCacheInitialSize = keySetInitialSize
		result.JobCacheFinalSize = keySetInitialSize
		utils.Debug(fmt.Sprintf("Job ID cache contains %d entries prior to scraping", keySetInitialSize))
	} else {
		utils.Debug("Skipping job ID file cache for this run")
	}

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
	scraper.ScrapeJobs(ctx, cfg.DefaultQuery, jobsChan, stats)
	processingWg.Wait()

	if cfg.UseJobIDFile {
		keySet = scraper.GetProcessedIDs()
		if err := dynamoService.WriteJobIdsToFile(cfg.Filename, keySet); err != nil {
			return nil, fmt.Errorf("write job ids: %w", err)
		}
		if cfg.UseS3JobIDFile && s3Service != nil {
			utils.Debug(fmt.Sprintf("Uploading %d job IDs to s3://%s/%s", len(keySet), cfg.JobIDsBucket, cfg.JobIDsS3Key))
			if err := uploadJobIDCache(ctx, s3Service, cfg.JobIDsBucket, cfg.JobIDsS3Key, cfg.Filename); err != nil {
				return nil, err
			}
			result.JobCacheS3Bucket = cfg.JobIDsBucket
			result.JobCacheS3Key = cfg.JobIDsS3Key
		}
		keySetFinalSize := len(keySet)
		result.JobCacheFinalSize = keySetFinalSize
		result.JobsAddedToCache = keySetFinalSize - keySetInitialSize
		utils.Debug(fmt.Sprintf("💰Jobs added to cache: %d", keySetFinalSize-keySetInitialSize))
		utils.Debug(fmt.Sprintf("Job ID cache now contains %d entries", keySetFinalSize))
	}

	executionTime := time.Since(startTime)
	// print stats
	stats.PrintSummary(executionTime)
	result.ExecutionTime = executionTime
	result.Stats = stats.Snapshot()

	return result, nil
}

func processAndSendJobs(ctx context.Context, jobsChan <-chan models.Job, stats *models.JobStats, cfg config.Config,
	parser services.ParserClient, dynamoService services.DynamoDBClient) {
	sem := make(chan struct{}, cfg.MaxConcurrency)
	var wg sync.WaitGroup

	for job := range jobsChan {
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

func downloadJobIDCache(ctx context.Context, s3Client services.S3Client, bucket, key, filename string) error {
	if err := s3Client.DownloadFile(ctx, bucket, key, filename); err != nil {
		var noKey *s3types.NoSuchKey
		if errors.As(err, &noKey) {
			utils.Debug(fmt.Sprintf("No existing job ID cache at s3://%s/%s; starting fresh", bucket, key))
			return nil
		}
		return fmt.Errorf("download job id cache: %w", err)
	}
	utils.Debug(fmt.Sprintf("Downloaded job ID cache from s3://%s/%s", bucket, key))
	return nil
}

func uploadJobIDCache(ctx context.Context, s3Client services.S3Client, bucket, key, filename string) error {
	if err := s3Client.UploadFile(ctx, bucket, key, filename); err != nil {
		return fmt.Errorf("upload job id cache: %w", err)
	}
	utils.Debug(fmt.Sprintf("Uploaded job ID cache to s3://%s/%s", bucket, key))
	return nil
}
