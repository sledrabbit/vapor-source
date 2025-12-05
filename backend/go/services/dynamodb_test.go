package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"gopher-source/models"
)

func TestDynamoPutJobHandlesDuplicate(t *testing.T) {
	var putCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "DynamoDB_20120810.PutItem":
			atomic.AddInt32(&putCount, 1)
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"ConditionalCheckFailedException"}`)
		default:
			t.Fatalf("unexpected target %s", r.Header.Get("X-Amz-Target"))
		}
	}))
	defer server.Close()

	client := newTestDynamoClient(server.URL)
	err := client.PutJob(context.Background(), &testJob)
	if err != nil {
		t.Fatalf("expected duplicate to be ignored, got %v", err)
	}
	if atomic.LoadInt32(&putCount) != 1 {
		t.Fatalf("expected 1 PutItem call, got %d", putCount)
	}
}

func TestDynamoGetAllJobIds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "DynamoDB_20120810.Scan":
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
			fmt.Fprint(w, `{"Items":[{"JobId":{"S":"abc"}},{"JobId":{"S":"def"}}]}`)
		case "DynamoDB_20120810.PutItem":
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
			fmt.Fprint(w, `{}`)
		default:
			t.Fatalf("unexpected target %s", r.Header.Get("X-Amz-Target"))
		}
	}))
	defer server.Close()

	client := newTestDynamoClient(server.URL)
	jobIDs, err := client.GetAllJobIds(context.Background())
	if err != nil {
		t.Fatalf("GetAllJobIds returned error: %v", err)
	}
	if len(jobIDs) != 2 || !jobIDs["abc"] || !jobIDs["def"] {
		t.Fatalf("unexpected job IDs: %+v", jobIDs)
	}
}

func TestWriteJobIdsToFile(t *testing.T) {
	client := &dynamoDBClientImpl{}
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "ids.txt")

	err := client.WriteJobIdsToFile(outFile, map[string]bool{"a": true, "b": true})
	if err != nil {
		t.Fatalf("WriteJobIdsToFile returned error: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	contents := string(data)
	if len(contents) == 0 || !(containsLine(contents, "a") && containsLine(contents, "b")) {
		t.Fatalf("unexpected file contents: %q", contents)
	}
}

func newTestDynamoClient(endpoint string) *dynamoDBClientImpl {
	cfg := aws.Config{
		Region:      "us-west-2",
		Credentials: aws.AnonymousCredentials{},
	}
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})
	return &dynamoDBClientImpl{
		client:    client,
		tableName: "Jobs",
	}
}

var testJob = models.Job{
	JobId:   "1",
	Title:   "Engineer",
	Company: "Acme",
}

func containsLine(contents, target string) bool {
	for _, line := range strings.Split(strings.TrimSpace(contents), "\n") {
		if strings.TrimSpace(line) == target {
			return true
		}
	}
	return false
}

func TestQueryJobsByPostedDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "DynamoDB_20120810.Query":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			defer r.Body.Close()
			payload := string(body)
			if !strings.Contains(payload, postedDateIndexName) {
				t.Fatalf("expected index name %s in payload %s", postedDateIndexName, payload)
			}
			if !strings.Contains(payload, "PostedDate = :date") {
				t.Fatalf("expected equality expression in payload %s", payload)
			}
			if !strings.Contains(payload, "2025-11-04") {
				t.Fatalf("expected requested date in payload %s", payload)
			}
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
			fmt.Fprint(w, `{"Items":[{"JobId":{"S":"1"},"Title":{"S":"Engineer"},"PostedDate":{"S":"2025-11-04"}}]}`)
		default:
			t.Fatalf("unexpected target %s", r.Header.Get("X-Amz-Target"))
		}
	}))
	defer server.Close()

	client := newTestDynamoClient(server.URL)
	jobs, err := client.QueryJobsByPostedDate(context.Background(), "2025-11-04")
	if err != nil {
		t.Fatalf("QueryJobsByPostedDate returned error: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].JobId != "1" || jobs[0].PostedDate != "2025-11-04" {
		t.Fatalf("unexpected job returned: %+v", jobs[0])
	}
}

func TestQueryJobsByDateRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("X-Amz-Target") {
		case "DynamoDB_20120810.Query":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			defer r.Body.Close()
			payload := string(body)
			if !strings.Contains(payload, "BETWEEN :start AND :end") {
				t.Fatalf("expected BETWEEN expression in payload %s", payload)
			}
			if !strings.Contains(payload, "2025-11-01") || !strings.Contains(payload, "2025-11-07") {
				t.Fatalf("expected start and end dates in payload %s", payload)
			}
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
			fmt.Fprint(w, `{"Items":[{"JobId":{"S":"1"},"PostedDate":{"S":"2025-11-01"}},{"JobId":{"S":"2"},"PostedDate":{"S":"2025-11-05"}}]}`)
		default:
			t.Fatalf("unexpected target %s", r.Header.Get("X-Amz-Target"))
		}
	}))
	defer server.Close()

	client := newTestDynamoClient(server.URL)
	jobs, err := client.QueryJobsByDateRange(context.Background(), "2025-11-01", "2025-11-07")
	if err != nil {
		t.Fatalf("QueryJobsByDateRange returned error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].JobId != "1" || jobs[1].JobId != "2" {
		t.Fatalf("unexpected jobs returned: %+v", jobs)
	}
}
