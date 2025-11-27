package services

import (
	"context"
	"fmt"
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
