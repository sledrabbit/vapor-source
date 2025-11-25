package main

import (
	"context"
	"log"

	"gopher-source/config"
	"gopher-source/internal/app"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg.UseJobIDFile = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := app.Run(ctx, cfg)
	if err != nil {
		log.Fatalf("Job pipeline failed: %v", err)
	}
	log.Printf("Job pipeline completed in %.2f seconds. Jobs added to cache: %d", result.ExecutionTime.Seconds(), result.JobsAddedToCache)
}
