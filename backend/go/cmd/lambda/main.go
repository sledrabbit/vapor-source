package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"

	"gopher-source/config"
	"gopher-source/internal/app"
)

func handler(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return app.Run(ctx, cfg)
}

func main() {
	lambda.Start(handler)
}
