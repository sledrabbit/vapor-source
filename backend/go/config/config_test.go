package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	cases := []struct {
		name     string
		setEnv   bool
		envValue string
		fallback string
		want     string
	}{
		{"missing", false, "", "default", "default"},
		{"custom", true, "real", "default", "real"},
		{"spacesOnly", true, "    ", "default", "default"},
		{"whiteSpace", true, "\n\n\t", "default", "default"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			const key = "OPENAI_API_KEY"

			if tc.setEnv {
				t.Setenv(key, tc.envValue)
			} else {
				t.Setenv(key, "")
				if err := os.Unsetenv(key); err != nil {
					t.Fatalf("failed to unset %s: %v", key, err)
				}
			}

			if got := getEnvOrDefault(key, tc.fallback); got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestNormalizeBoolToString(t *testing.T) {
	cases := []struct {
		name       string
		truthValue string
		fallback   bool
		want       string
	}{
		{"1", "true", false, "true"},
		{"true", "true", false, "true"},
		{"yes", "true", false, "true"},
		{"y", "true", false, "true"},
		{"on", "true", false, "true"},
		{"0", "false", true, "false"},
		{"false", "false", true, "false"},
		{"no", "false", true, "false"},
		{"n", "false", true, "false"},
		{"off", "false", true, "false"},
		{"whitespace", "\nfalse", true, "false"},
		{"nonTruthTrueFallback", "asdf", true, "true"},
		{"nonTruthFalseFallback", "asdf", false, "false"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			if got := normalizeBoolString(tc.truthValue, tc.fallback); got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

func TestRunningInLambda(t *testing.T) {
	cases := []struct {
		name string
		envs map[string]string
		want bool
	}{
		{"noneSet", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "", "LAMBDA_TASK_ROOT": ""}, false},
		{"functionNameSet", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "handler", "LAMBDA_TASK_ROOT": ""}, true},
		{"taskRootSet", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "", "LAMBDA_TASK_ROOT": "/var/task"}, true},
		{"whitespace", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "   ", "LAMBDA_TASK_ROOT": ""}, false},
		{"bothSet", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "handler", "LAMBDA_TASK_ROOT": "/var/task"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, key := range []string{"AWS_LAMBDA_FUNCTION_NAME", "LAMBDA_TASK_ROOT"} {
				value, ok := tc.envs[key]
				if !ok {
					value = ""
				}
				t.Setenv(key, value)
			}

			if got := runningInLambda(); got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestDefaultJobIDsPath(t *testing.T) {
	cases := []struct {
		name     string
		lambda   bool
		expected string
	}{
		{"nonLambda", false, "job-ids.txt"},
		{"lambda", true, filepath.Join(os.TempDir(), "job-ids.txt")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.lambda {
				t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "handler")
				t.Setenv("LAMBDA_TASK_ROOT", "")
			} else {
				t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "")
				t.Setenv("LAMBDA_TASK_ROOT", "")
			}

			if got := defaultJobIDsPath(); got != tc.expected {
				t.Fatalf("want %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestLoadSuccess(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "")
	t.Setenv("LAMBDA_TASK_ROOT", "")
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("QUERY", "backend engineer")
	t.Setenv("JOB_IDS_PATH", "/tmp/job-ids.txt")
	t.Setenv("DEBUG_OUTPUT", "true")
	t.Setenv("API_DRY_RUN", "false")
	t.Setenv("USE_JOB_ID_FILE", "false")
	t.Setenv("USE_S3_JOB_ID_FILE", "true")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("DYNAMODB_TABLE_NAME", "JobsTest")
	t.Setenv("DYNAMODB_ENDPOINT", "http://localhost:9000")
	t.Setenv("JOB_IDS_BUCKET", "jobs-bucket")
	t.Setenv("JOB_IDS_S3_KEY", "ids.txt")
	t.Setenv("SNAPSHOT_BUCKET", "snapshot-bucket")
	t.Setenv("SNAPSHOT_S3_KEY", "snapshot-key.txt")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.OpenAIAPIKey != "test-key" {
		t.Fatalf("expected OpenAIAPIKey to be test-key, got %q", cfg.OpenAIAPIKey)
	}
	if cfg.Query != "backend engineer" || cfg.DefaultQuery != "backend engineer" {
		t.Fatalf("expected query fields to match custom value, got %q and %q", cfg.Query, cfg.DefaultQuery)
	}
	if cfg.DebugOutput != "true" {
		t.Fatalf("expected DebugOutput to be true, got %q", cfg.DebugOutput)
	}
	if cfg.ApiDryRun != "false" {
		t.Fatalf("expected ApiDryRun to be false, got %q", cfg.ApiDryRun)
	}
	if cfg.Filename != "/tmp/job-ids.txt" {
		t.Fatalf("expected Filename to reflect JOB_IDS_PATH, got %q", cfg.Filename)
	}
	if cfg.UseJobIDFile {
		t.Fatalf("expected UseJobIDFile to be false when override provided")
	}
	if !cfg.UseS3JobIDFile {
		t.Fatalf("expected UseS3JobIDFile to be true when override provided")
	}
	if cfg.AWSRegion != "us-east-1" {
		t.Fatalf("expected AWSRegion override, got %q", cfg.AWSRegion)
	}
	if cfg.DynamoTableName != "JobsTest" {
		t.Fatalf("expected DynamoTableName override, got %q", cfg.DynamoTableName)
	}
	if cfg.DynamoEndpoint != "http://localhost:9000" {
		t.Fatalf("expected DynamoEndpoint override, got %q", cfg.DynamoEndpoint)
	}
	if cfg.JobIDsBucket != "jobs-bucket" {
		t.Fatalf("expected JobIDsBucket override, got %q", cfg.JobIDsBucket)
	}
	if cfg.JobIDsS3Key != "ids.txt" {
		t.Fatalf("expected JobIDsS3Key override, got %q", cfg.JobIDsS3Key)
	}
	if cfg.SnapshotBucket != "snapshot-bucket" {
		t.Fatalf("expected SnapshotBucket override, got %q", cfg.SnapshotBucket)
	}
	if cfg.SnapshotS3Key != "snapshot-key.txt" {
		t.Fatalf("expected SnapshotS3Key override, got %q", cfg.SnapshotS3Key)
	}
}

func TestLoadRequiresAPIKeyWhenNotDryRun(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "")
	t.Setenv("LAMBDA_TASK_ROOT", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("API_DRY_RUN", "false")

	if _, err := Load(); err == nil {
		t.Fatal("expected error when API key missing and dry run disabled")
	} else if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("expected error to mention API key, got %v", err)
	}
}

func TestLoadAllowsDryRunWithoutAPIKey(t *testing.T) {
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "")
	t.Setenv("LAMBDA_TASK_ROOT", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("API_DRY_RUN", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ApiDryRun != "true" {
		t.Fatalf("expected ApiDryRun to be true, got %q", cfg.ApiDryRun)
	}
	if cfg.OpenAIAPIKey != "" {
		t.Fatalf("expected OpenAIAPIKey to remain empty, got %q", cfg.OpenAIAPIKey)
	}
}
