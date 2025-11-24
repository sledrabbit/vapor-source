package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"gopher-source/config"
	"gopher-source/internal/app"
)

type Request events.APIGatewayV2HTTPRequest
type Response events.APIGatewayV2HTTPResponse

type apiResponse struct {
	Message   string `json:"message"`
	Query     string `json:"query"`
	DebugMode bool   `json:"debugMode"`
	DryRun    bool   `json:"dryRunMode"`
}

func handler(ctx context.Context, event Request) (Response, error) {
	cfg, err := config.Load()
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("load config: %w", err))
	}

	applyRequestOverrides(cfg, event.QueryStringParameters)

	if err := app.Run(ctx, cfg); err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("run app: %w", err))
	}

	payload := apiResponse{
		Message:   "Job processing completed",
		Query:     cfg.Query,
		DebugMode: parseBool(cfg.DebugOutput),
		DryRun:    parseBool(cfg.ApiDryRun),
	}

	return jsonResponse(http.StatusOK, payload), nil
}

func applyRequestOverrides(cfg *config.Config, params map[string]string) {
	if params == nil {
		return
	}

	if q := strings.TrimSpace(params["query"]); q != "" {
		cfg.Query = q
	}

	if debug := strings.TrimSpace(params["debug"]); debug != "" {
		cfg.DebugOutput = boolToString(debug)
	}

	if dryRun := strings.TrimSpace(params["dryrun"]); dryRun != "" {
		cfg.ApiDryRun = boolToString(dryRun)
	}
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func boolToString(value string) string {
	if parseBool(value) {
		return "true"
	}
	return "false"
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

func main() {
	lambda.Start(handler)
}
