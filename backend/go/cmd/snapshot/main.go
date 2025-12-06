package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

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

	log.Printf("snapshot: fetched %d jobs between %s and %s", len(sortedJobs), startDate, endDate)
	if len(sortedJobs) == 0 {
		log.Printf("snapshot: no jobs found for requested date range; exiting")
		return jsonResponse(http.StatusOK, apiResponse{Message: "Snapshot completed - no jobs for requested date(s)"}), nil
	}

	// upload snapshot files to s3
	groupedJobs := groupJobsByPostedDate(sortedJobs, startDate)
	filesWritten, err := writeAndUploadSnapshots(ctx, groupedJobs, cfg, s3Service)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, err)
	}
	if err := updateSnapshotManifest(ctx, cfg, s3Service, filesWritten); err != nil {
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

func writeAndUploadSnapshots(ctx context.Context, groups map[string][]models.Job, cfg *config.Config, s3Service services.S3Client) ([]snapshotFileMetadata, error) {
	var dates []string
	for date := range groups {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	tempDir := os.TempDir()
	var written []snapshotFileMetadata
	for _, date := range dates {
		filename := fmt.Sprintf("%s.jsonl", date)
		localPath := filepath.Join(tempDir, filename)

		if err := s3Service.WriteJobsToJSONLFile(localPath, groups[date]); err != nil {
			return nil, fmt.Errorf("write jobs for %s: %w", date, err)
		}

		objectKey := snapshotObjectKey(cfg, filename)
		if err := s3Service.UploadFile(ctx, cfg.SnapshotBucket, objectKey, localPath); err != nil {
			return nil, fmt.Errorf("upload snapshot %s: %w", date, err)
		}
		log.Printf("snapshot: wrote %d jobs to s3://%s/%s", len(groups[date]), cfg.SnapshotBucket, objectKey)
		written = append(written, snapshotFileMetadata{
			Date:     date,
			Key:      objectKey,
			JobCount: len(groups[date]),
		})
		_ = os.Remove(localPath)
	}
	return written, nil
}

func updateSnapshotManifest(ctx context.Context, cfg *config.Config, s3Service services.S3Client, files []snapshotFileMetadata) error {
	if len(files) == 0 {
		log.Printf("snapshot: no files written; skipping manifest update")
		return nil
	}

	manifestKey := snapshotManifestKey(cfg)
	existing, err := loadSnapshotManifest(ctx, cfg, s3Service, manifestKey)
	if err != nil {
		return fmt.Errorf("load snapshot manifest: %w", err)
	}

	entries := make(map[string]snapshotManifestEntry)
	for _, entry := range existing {
		entries[entry.Date] = entry
	}

	for _, file := range files {
		entries[file.Date] = snapshotManifestEntry{
			Date:      file.Date,
			Key:       file.Key,
			JobCount:  file.JobCount,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}
	}

	manifest := make([]snapshotManifestEntry, 0, len(entries))
	for _, entry := range entries {
		manifest = append(manifest, entry)
	}
	sort.Slice(manifest, func(i, j int) bool {
		if manifest[i].Date == manifest[j].Date {
			return manifest[i].UpdatedAt > manifest[j].UpdatedAt
		}
		return manifest[i].Date > manifest[j].Date
	})

	if err := writeSnapshotManifest(ctx, cfg, s3Service, manifestKey, manifest); err != nil {
		return fmt.Errorf("write snapshot manifest: %w", err)
	}

	log.Printf("snapshot: manifest updated (%d entries) at s3://%s/%s", len(manifest), cfg.SnapshotBucket, manifestKey)
	return nil
}

func loadSnapshotManifest(ctx context.Context, cfg *config.Config, s3Service services.S3Client, manifestKey string) ([]snapshotManifestEntry, error) {
	tmpFile, err := os.CreateTemp("", "snapshot-manifest-*.json")
	if err != nil {
		return nil, fmt.Errorf("create manifest temp file: %w", err)
	}
	path := tmpFile.Name()
	_ = tmpFile.Close()
	defer os.Remove(path)

	if err := s3Service.DownloadFile(ctx, cfg.SnapshotBucket, manifestKey, path); err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			return nil, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	content := bytes.TrimSpace(data)
	if len(content) == 0 {
		return nil, nil
	}

	var entries []snapshotManifestEntry
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}
	return entries, nil
}

func writeSnapshotManifest(ctx context.Context, cfg *config.Config, s3Service services.S3Client, manifestKey string, entries []snapshotManifestEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "snapshot-manifest-*.json")
	if err != nil {
		return fmt.Errorf("create manifest file: %w", err)
	}
	path := tmpFile.Name()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write manifest temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close manifest temp file: %w", err)
	}
	defer os.Remove(path)

	if err := s3Service.UploadFile(ctx, cfg.SnapshotBucket, manifestKey, path); err != nil {
		return err
	}
	return nil
}

func snapshotObjectKey(cfg *config.Config, filename string) string {
	prefix := strings.Trim(strings.TrimSpace(cfg.SnapshotS3Key), "/")
	if prefix == "" {
		return filename
	}
	return fmt.Sprintf("%s/%s", prefix, filename)
}

func snapshotManifestKey(cfg *config.Config) string {
	prefix := strings.Trim(strings.TrimSpace(cfg.SnapshotS3Key), "/")
	if prefix == "" {
		return "snapshot-manifest.json"
	}
	return fmt.Sprintf("%s/snapshot-manifest.json", prefix)
}

type snapshotFileMetadata struct {
	Date     string
	Key      string
	JobCount int
}

type snapshotManifestEntry struct {
	Date      string `json:"date"`
	Key       string `json:"key"`
	JobCount  int    `json:"jobCount"`
	UpdatedAt string `json:"updatedAt"`
}
