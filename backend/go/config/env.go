package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func GetEnv(name string) (string, error) {
	if err := godotenv.Load(".env"); err != nil {
		return "", fmt.Errorf("Failed to load environment variable.")
	}
	apiKey := os.Getenv(name)

	if apiKey == "" {
		return "", fmt.Errorf("Environment variable is not set.")
	}
	return apiKey, nil
}
