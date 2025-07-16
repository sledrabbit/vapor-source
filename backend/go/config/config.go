package config

import (
	"fmt"
	"os"
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
}

func Load() (*Config, error) {
	if err := godotenv.Load(".env"); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	query := os.Getenv("QUERY")
	if query == "" {
		query = "software developer"
	}

	debugOutput := os.Getenv("DEBUG_OUTPUT")
	if debugOutput == "" {
		return nil, fmt.Errorf("DEBUG_OUTPUT environment variable is not set")
	}

	apiDryRun := os.Getenv("API_DRY_RUN")
	if apiDryRun == "" {
		return nil, fmt.Errorf("API_DRY_RUN environment variable is not set")
	}

	return &Config{
		MaxPages:       5,
		BaseURL:        "https://seeker.worksourcewa.com/",
		RequestDelay:   1 * time.Nanosecond,
		OpenAIAPIKey:   apiKey,
		Query:          query,
		DebugOutput:    debugOutput,
		ApiDryRun:      apiDryRun,
		MaxConcurrency: 25,
		DefaultQuery:   "software developer",
	}, nil
}
