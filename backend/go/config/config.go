package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	MaxPages       int
	BaseURL        string
	RequestDelay   time.Duration
	OpenAIAPIKey   string
	Query          string
	DebugOutput    string
	ApiDryRun      string
	MaxConcurrency int
	DefaultQuery   string
	Filename       string
}

var (
	envLoadOnce sync.Once
	envLoadErr  error
)

func init() {
	_ = EnsureEnvLoaded()
}

func Load() (*Config, error) {
	if err := EnsureEnvLoaded(); err != nil {
		return nil, err
	}

	query := getEnvOrDefault("QUERY", "software developer")
	jobIDsPath := getEnvOrDefault("JOB_IDS_PATH", defaultJobIDsPath())

	return &Config{
		MaxPages:       2,
		BaseURL:        "https://seeker.worksourcewa.com/",
		RequestDelay:   1 * time.Nanosecond,
		OpenAIAPIKey:   strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		Query:          query,
		DebugOutput:    getBoolEnv("DEBUG_OUTPUT", false),
		ApiDryRun:      getBoolEnv("API_DRY_RUN", false),
		MaxConcurrency: 25,
		DefaultQuery:   query,
		Filename:       jobIDsPath,
	}, nil
}

func EnsureEnvLoaded() error {
	envLoadOnce.Do(func() {
		envLoadErr = loadDotEnv()
	})
	return envLoadErr
}

func loadDotEnv() error {
	if err := godotenv.Overload(".env"); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load .env file: %w", err)
	}
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) string {
	if value, ok := os.LookupEnv(key); ok {
		return normalizeBoolString(value, fallback)
	}
	return boolToString(fallback)
}

func normalizeBoolString(value string, fallback bool) string {
	normalized := strings.ToLower(strings.Trim(value, " \t\n\r,"))
	switch normalized {
	case "1", "true", "yes", "y", "on":
		return "true"
	case "0", "false", "no", "n", "off":
		return "false"
	}
	return boolToString(fallback)
}

func boolToString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func defaultJobIDsPath() string {
	if runningInLambda() {
		return filepath.Join(os.TempDir(), "job-ids.txt")
	}
	return "job-ids.txt"
}

func runningInLambda() bool {
	return strings.TrimSpace(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")) != "" ||
		strings.TrimSpace(os.Getenv("LAMBDA_TASK_ROOT")) != ""
}
