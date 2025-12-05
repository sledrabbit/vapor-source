package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"gopher-source/config"
	"gopher-source/services"
)

type Request events.APIGatewayV2HTTPRequest
type Response events.APIGatewayV2HTTPResponse

type apiResponse struct {
	Message string `json:"message"`
}

func handler(ctx context.Context) (Response, error) {
	start := time.Now()
	cfg, err := config.Load()
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("load config: %w", err))
	}

	awscfg, err := services.NewDynamoConfig(ctx, cfg.AWSRegion)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("load aws config: %w", err))
	}

	dynamoService := services.NewDynamoService(awscfg, cfg.DynamoTableName, cfg.DynamoEndpoint)

	_, err = dynamoService.QueryJobsByPostedDate(ctx, time.Now().Format("2006-01-02"))
	if err != nil {
		return errorResponse(http.StatusInternalServerError, fmt.Errorf("query DynamoDB: %w", err))
	}

	functionDuration := time.Since(start)
	print(functionDuration)

	payload := apiResponse{
		Message: "Snapshot processing completed",
	}

	return jsonResponse(http.StatusOK, payload), nil
}

func main() {
	lambda.Start(handler)
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
