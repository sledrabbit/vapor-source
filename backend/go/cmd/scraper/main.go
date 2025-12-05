package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"

	"gopher-source/config"
	"gopher-source/internal/app"
)

type Request events.APIGatewayV2HTTPRequest
type Response events.APIGatewayV2HTTPResponse

type apiResponse struct {
	Message       string               `json:"message"`
	Query         string               `json:"query"`
	DebugMode     bool                 `json:"debugMode"`
	DryRun        bool                 `json:"dryRunMode"`
	Stats         jobStatsPayload      `json:"stats"`
	JobCache      jobCachePayload      `json:"jobCache"`
	LambdaMetrics lambdaMetricsPayload `json:"lambdaMetrics"`
}

type jobStatsPayload struct {
	TotalJobs            int64   `json:"totalJobsScraped"`
	ProcessedJobs        int64   `json:"jobsProcessed"`
	JobsSkipped          int64   `json:"jobsSkipped"`
	UnrelatedJobs        int64   `json:"unrelatedJobs"`
	SuccessfullyParsed   int64   `json:"successfullyParsed"`
	FailedToParse        int64   `json:"failedToParse"`
	SuccessRate          float64 `json:"successRate"`
	ExecutionTimeSeconds float64 `json:"executionTimeSeconds"`
	JobsPerSecond        float64 `json:"jobsPerSecond"`
}

type jobCachePayload struct {
	Enabled        bool   `json:"enabled"`
	InitialEntries int    `json:"initialEntries"`
	FinalEntries   int    `json:"finalEntries"`
	JobsAdded      int    `json:"jobsAdded"`
	Bucket         string `json:"bucket,omitempty"`
	ObjectKey      string `json:"objectKey,omitempty"`
}

type lambdaMetricsPayload struct {
	FunctionDurationMs  int64   `json:"functionDurationMs"`
	BilledDurationMs    int64   `json:"billedDurationMs"`
	MemoryLimitMB       int     `json:"memoryLimitMB"`
	EstimatedMemoryUsed float64 `json:"estimatedMemoryUsedMB"`
	PipelineDurationMs  int64   `json:"pipelineDurationMs"`
}

func handler(ctx context.Context, event Request) (Response, error) {
	start := time.Now()
	cfg, err := config.Load()
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("load config: %w", err))
	}

	applyRequestOverrides(cfg, event.QueryStringParameters)

	runResult, err := app.Run(ctx, cfg)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("run app: %w", err))
	}
	functionDuration := time.Since(start)

	payload := apiResponse{
		Message:       "Job processing completed",
		Query:         cfg.Query,
		DebugMode:     parseBool(cfg.DebugOutput),
		DryRun:        parseBool(cfg.ApiDryRun),
		Stats:         buildJobStatsPayload(runResult),
		JobCache:      buildJobCachePayload(runResult),
		LambdaMetrics: buildLambdaMetrics(functionDuration, runResult.ExecutionTime),
	}

	return jsonResponse(http.StatusOK, payload), nil
}

// to control knobs via query parameters
func applyRequestOverrides(cfg *config.Config, params map[string]string) {
	if params == nil {
		return
	}

	if q := strings.TrimSpace(params["query"]); q != "" {
		cfg.Query = q
	}

	if pages := strings.TrimSpace(params["pages"]); pages != "" {
		if maxPages, err := strconv.Atoi(pages); err == nil && maxPages > 0 {
			cfg.MaxPages = maxPages
		}
	}

	if debug := strings.TrimSpace(params["debug"]); debug != "" {
		cfg.DebugOutput = boolToString(debug)
	}

	if dryRun := strings.TrimSpace(params["dryrun"]); dryRun != "" {
		cfg.ApiDryRun = boolToString(dryRun)
	}
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func boolToString(value string) string {
	if parseBool(value) {
		return "true"
	}
	return "false"
}

func buildJobStatsPayload(result *app.RunResult) jobStatsPayload {
	stats := result.Stats
	executionSeconds := result.ExecutionTime.Seconds()
	jobsHandled := stats.ProcessedJobs + stats.SkippedJobs
	var successRate float64
	if stats.ProcessedJobs > 0 {
		successRate = float64(stats.SuccessfulJobs) / float64(stats.ProcessedJobs) * 100
	}
	var jobsPerSecond float64
	if executionSeconds > 0 {
		jobsPerSecond = float64(jobsHandled) / executionSeconds
	}

	return jobStatsPayload{
		TotalJobs:            stats.TotalJobs,
		ProcessedJobs:        stats.ProcessedJobs,
		JobsSkipped:          stats.SkippedJobs,
		UnrelatedJobs:        stats.UnrelatedJobs,
		SuccessfullyParsed:   stats.SuccessfulJobs,
		FailedToParse:        stats.FailedJobs,
		SuccessRate:          round(successRate, 2),
		ExecutionTimeSeconds: round(executionSeconds, 2),
		JobsPerSecond:        round(jobsPerSecond, 4),
	}
}

func buildJobCachePayload(result *app.RunResult) jobCachePayload {
	return jobCachePayload{
		Enabled:        result.JobCacheEnabled,
		InitialEntries: result.JobCacheInitialSize,
		FinalEntries:   result.JobCacheFinalSize,
		JobsAdded:      result.JobsAddedToCache,
		Bucket:         result.JobCacheS3Bucket,
		ObjectKey:      result.JobCacheS3Key,
	}
}

func buildLambdaMetrics(functionDuration, pipelineDuration time.Duration) lambdaMetricsPayload {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	memoryLimit := lambdacontext.MemoryLimitInMB
	billedMs := int64(math.Ceil(functionDuration.Seconds() * 1000))
	return lambdaMetricsPayload{
		FunctionDurationMs:  functionDuration.Milliseconds(),
		BilledDurationMs:    billedMs,
		MemoryLimitMB:       memoryLimit,
		EstimatedMemoryUsed: round(bytesToMB(memStats.Alloc), 2),
		PipelineDurationMs:  pipelineDuration.Milliseconds(),
	}
}

func bytesToMB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}

func round(value float64, precision int) float64 {
	multiplier := math.Pow(10, float64(precision))
	return math.Round(value*multiplier) / multiplier
}

func jsonResponse(status int, payload interface{}) Response {
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: fmt.Sprintf(`{"message":"%s"}`, err.Error()),
		}
	}

	return Response{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}
}

func errorResponse(status int, err error) (Response, error) {
	payload := map[string]string{
		"message": err.Error(),
	}

	return jsonResponse(status, payload), nil
}

func main() {
	lambda.Start(handler)
}
