package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"gopher-source/config"
	"gopher-source/models"
	"gopher-source/services"
)

type Request events.APIGatewayV2HTTPRequest
type Response events.APIGatewayV2HTTPResponse

type apiResponse struct {
	Message string `json:"message"`
}

func handler(ctx context.Context) (Response, error) {
	start := time.Now()
	cfg, err := config.Load()
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("load config: %w", err))
	}

	awscfg, err := services.NewDynamoConfig(ctx, cfg.AWSRegion)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("load aws config: %w", err))
	}

	dynamoService := services.NewDynamoService(awscfg, cfg.DynamoTableName, cfg.DynamoEndpoint)
	s3Service := services.NewS3Service(awscfg)

	startDate, endDate, err := determineDateRange(cfg)
	if err != nil {
		return errorResponse(http.StatusBadRequest, err)
	}

	// get sorted jobs in descending order
	var sortedJobs []models.Job
	dateCursor, _ := time.Parse("2006-01-02", startDate)
	endTime, _ := time.Parse("2006-01-02", endDate)
	for !dateCursor.After(endTime) {
		dateStr := dateCursor.Format("2006-01-02")
		dailyJobs, queryErr := dynamoService.QueryJobsByPostedDate(ctx, dateStr)
		if queryErr != nil {
			return errorResponse(http.StatusInternalServerError, fmt.Errorf("query DynamoDB for %s: %w", dateStr, queryErr))
		}
		sortedJobs = append(sortedJobs, dailyJobs...)
		dateCursor = dateCursor.AddDate(0, 0, 1)
	}

	if len(sortedJobs) == 0 {
		return jsonResponse(http.StatusOK, apiResponse{Message: "Snapshot completed - no jobs for requested date(s)"}), nil
	}

	// upload snapshot files to s3
	groupedJobs := groupJobsByPostedDate(sortedJobs, startDate)
	if err := writeAndUploadSnapshots(ctx, groupedJobs, cfg, s3Service); err != nil {
		return errorResponse(http.StatusInternalServerError, err)
	}

	functionDuration := time.Since(start)
	print(functionDuration)

	payload := apiResponse{
		Message: "Snapshot processing completed",
	}

	return jsonResponse(http.StatusOK, payload), nil
}

func main() {
	lambda.Start(handler)
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

// determineDateRange returns the start/end date range using optional env overrides.
func determineDateRange(cfg *config.Config) (string, string, error) {
	const layout = "2006-01-02"
	start := strings.TrimSpace(cfg.SnapshotStartDate)
	end := strings.TrimSpace(cfg.SnapshotEndDate)

	if start == "" && end == "" {
		today := time.Now().UTC().Format(layout)
		return today, today, nil
	}
	if start == "" {
		start = end
	}
	if end == "" {
		end = start
	}
	startTime, err := time.Parse(layout, start)
	if err != nil {
		return "", "", fmt.Errorf("invalid SNAPSHOT_START_DATE: %w", err)
	}
	endTime, err := time.Parse(layout, end)
	if err != nil {
		return "", "", fmt.Errorf("invalid SNAPSHOT_END_DATE: %w", err)
	}
	if endTime.Before(startTime) {
		return "", "", fmt.Errorf("SNAPSHOT_END_DATE must be on or after SNAPSHOT_START_DATE")
	}
	return startTime.Format(layout), endTime.Format(layout), nil
}

// groups and prepares jobs by posted date for s3 upload
func groupJobsByPostedDate(jobs []models.Job, fallbackDate string) map[string][]models.Job {
	grouped := make(map[string][]models.Job)
	for _, job := range jobs {
		date := strings.TrimSpace(job.PostedDate)
		if date == "" {
			date = fallbackDate
		}
		grouped[date] = append(grouped[date], job)
	}
	return grouped
}

func writeAndUploadSnapshots(ctx context.Context, groups map[string][]models.Job, cfg *config.Config, s3Service services.S3Client) error {
	var dates []string
	for date := range groups {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	tempDir := os.TempDir()
	for _, date := range dates {
		filename := fmt.Sprintf("%s.jsonl", date)
		localPath := filepath.Join(tempDir, filename)

		if err := s3Service.WriteJobsToJSONLFile(localPath, groups[date]); err != nil {
			return fmt.Errorf("write jobs for %s: %w", date, err)
		}

		objectKey := filename
		if trimmed := strings.TrimSpace(cfg.SnapshotS3Key); trimmed != "" {
			objectKey = fmt.Sprintf("%s/%s", strings.TrimSuffix(trimmed, "/"), filename)
		}

		if err := s3Service.UploadFile(ctx, cfg.SnapshotBucket, objectKey, localPath); err != nil {
			return fmt.Errorf("upload snapshot %s: %w", date, err)
		}
		_ = os.Remove(localPath)
	}
	return nil
}
