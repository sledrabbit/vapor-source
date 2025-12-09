package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"gopher-source/models"
)

func TestS3UploadFileAndWait(t *testing.T) {
	stub := newS3Stub()
	server := httptest.NewServer(stub)
	defer server.Close()

	client := newTestS3Client(server.URL)
	tmpFile := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	err := client.UploadFile(context.Background(), "bucket", "object", tmpFile)
	if err != nil {
		t.Fatalf("UploadFile returned error: %v", err)
	}
	if got := stub.getObject("bucket", "object"); string(got) != "hello world" {
		t.Fatalf("expected stored object, got %q", string(got))
	}
}

func TestS3DownloadFile(t *testing.T) {
	stub := newS3Stub()
	stub.putObject("bucket", "object", []byte("downloaded data"))
	server := httptest.NewServer(stub)
	defer server.Close()

	client := newTestS3Client(server.URL)
	outFile := filepath.Join(t.TempDir(), "download.txt")
	if err := client.DownloadFile(context.Background(), "bucket", "object", outFile); err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != "downloaded data" {
		t.Fatalf("unexpected file contents: %q", string(data))
	}
}

func newTestS3Client(endpoint string) *s3ClientImpl {
	cfg := aws.Config{
		Region:      "us-west-2",
		Credentials: aws.AnonymousCredentials{},
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &s3ClientImpl{client: client}
}

type s3Stub struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newS3Stub() *s3Stub {
	return &s3Stub{objects: make(map[string][]byte)}
}

func (s *s3Stub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucket, key := parsePath(r.URL.Path)
	resource := bucket + "/" + key
	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		s.putObject(bucket, key, body)
		w.WriteHeader(http.StatusOK)
	case http.MethodHead:
		if _, ok := s.objects[resource]; ok {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `<Error><Code>NoSuchKey</Code></Error>`)
		}
	case http.MethodGet:
		if data, ok := s.objects[resource]; ok {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `<Error><Code>NoSuchKey</Code></Error>`)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *s3Stub) putObject(bucket, key string, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resource := bucket + "/" + key
	s.objects[resource] = append([]byte(nil), body...)
}

func (s *s3Stub) getObject(bucket, key string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	resource := bucket + "/" + key
	return append([]byte(nil), s.objects[resource]...)
}

func parsePath(path string) (bucket, key string) {
	trimmed := strings.TrimPrefix(path, "/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func TestWriteJobsToJSONLFile_Success(t *testing.T) {
	svc := &s3ClientImpl{}
	filename := filepath.Join(t.TempDir(), "jobs.jsonl")
	jobs := []models.Job{
		{JobId: "1", Title: "One", PostedDate: "2025-01-01", PostedTime: "2025-01-01T00:00:00Z"},
		{JobId: "2", Title: "Two", PostedDate: "2025-01-01", PostedTime: "2025-01-01T01:00:00Z"},
	}

	if err := svc.WriteJobsToJSONLFile(filename, jobs); err != nil {
		t.Fatalf("WriteJobsToJSONLFile returned error: %v", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open jsonl file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var got []models.Job
	for scanner.Scan() {
		var job models.Job
		if err := json.Unmarshal(scanner.Bytes(), &job); err != nil {
			t.Fatalf("unmarshal job line: %v", err)
		}
		got = append(got, job)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	if !reflect.DeepEqual(got, jobs) {
		t.Fatalf("jobs mismatch: got %#v want %#v", got, jobs)
	}
}

func TestWriteJobsToJSONLFile_CreateError(t *testing.T) {
	svc := &s3ClientImpl{}
	filename := filepath.Join(t.TempDir(), "missing", "jobs.jsonl")
	if err := svc.WriteJobsToJSONLFile(filename, nil); err == nil {
		t.Fatalf("expected error for unwritable path, got nil")
	}
}

func TestContentTypeForKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		key      string
		expected string
	}{
		{"snapshot.json", "application/json"},
		{"snapshot.JSONL", "application/json"},
		{"snapshot.jsonl.gz", "application/json"},
		{"notes.txt", "text/plain"},
		{"binary.bin", ""},
		{"", ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.key, func(t *testing.T) {
			if got := contentTypeForKey(tt.key); got != tt.expected {
				t.Fatalf("contentTypeForKey(%q) = %q, want %q", tt.key, got, tt.expected)
			}
		})
	}
}
