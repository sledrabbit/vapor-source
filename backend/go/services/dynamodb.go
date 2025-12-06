package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"gopher-source/models"
	"gopher-source/utils"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient interface {
	PutJob(ctx context.Context, job *models.Job) error
	QueryJobsByPostedDate(ctx context.Context, date string) ([]models.Job, error)
	GetAllJobIds(ctx context.Context) (map[string]bool, error)
	WriteJobIdsToFile(filename string, keySet map[string]bool) error
}

type dynamoDBClientImpl struct {
	client    *dynamodb.Client
	tableName string
}

const postedDateIndexName = "PostedDate-Index"

func NewDynamoService(cfg aws.Config, tableName, endpoint string) DynamoDBClient {
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if strings.TrimSpace(endpoint) != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &dynamoDBClientImpl{client: client, tableName: tableName}
}

func NewDynamoConfig(ctx context.Context, region string) (aws.Config, error) {
	loaders := []func(*awscfg.LoadOptions) error{}
	if strings.TrimSpace(region) != "" {
		loaders = append(loaders, awscfg.WithRegion(region))
	}
	cfg, err := awscfg.LoadDefaultConfig(ctx, loaders...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws config: %w", err)
	}
	return cfg, nil
}

func (d *dynamoDBClientImpl) QueryJobsByPostedDate(ctx context.Context, date string) ([]models.Job, error) {
	date = strings.TrimSpace(date)
	if date == "" {
		return nil, fmt.Errorf("posted date is required")
	}

	values := map[string]types.AttributeValue{
		":date": &types.AttributeValueMemberS{Value: date},
	}

	return d.queryJobs(ctx, "PostedDate = :date", values)
}

func (d *dynamoDBClientImpl) queryJobs(ctx context.Context, keyCondition string, values map[string]types.AttributeValue) ([]models.Job, error) {
	if strings.TrimSpace(keyCondition) == "" {
		return nil, fmt.Errorf("key condition expression is required")
	}
	if d.client == nil {
		return nil, fmt.Errorf("dynamodb client is not initialized")
	}

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(d.tableName),
		IndexName:                 aws.String(postedDateIndexName),
		KeyConditionExpression:    aws.String(keyCondition),
		ExpressionAttributeValues: values,
		ScanIndexForward:          aws.Bool(false),
	}

	paginator := dynamodb.NewQueryPaginator(d.client, input)
	var jobs []models.Job
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("query jobs: %w", err)
		}
		if len(output.Items) == 0 {
			continue
		}
		var pageJobs []models.Job
		if err := attributevalue.UnmarshalListOfMaps(output.Items, &pageJobs); err != nil {
			return nil, fmt.Errorf("unmarshal jobs: %w", err)
		}
		jobs = append(jobs, pageJobs...)
	}

	return jobs, nil
}

func (d *dynamoDBClientImpl) PutJob(ctx context.Context, job *models.Job) error {
	cond := expression.Name(job.JobId).AttributeNotExists()
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return err
	}
	item, err := job.ToDynamoDBItem()
	if err != nil {
		return fmt.Errorf("failed to marshal job to DynamoDB item: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName:                 aws.String(d.tableName),
		Item:                      item,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	_, err = d.client.PutItem(ctx, input)
	if err != nil {
		var conditionalCheckErr *types.ConditionalCheckFailedException
		if errors.As(err, &conditionalCheckErr) {
			utils.Debug("\t⚠️  Item already exists, skipping")
			return nil
		}
		return fmt.Errorf("failed to put item: %w", err)
	}
	utils.Debug(fmt.Sprintf("\t📦 Post successful: for job %s", job.Title))
	return nil
}

func (d *dynamoDBClientImpl) GetAllJobIds(ctx context.Context) (map[string]bool, error) {
	jobIds := make(map[string]bool)

	proj := expression.NamesList(expression.Name("JobId"))
	expr, err := expression.NewBuilder().WithProjection(proj).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	input := &dynamodb.ScanInput{
		TableName:                aws.String(d.tableName),
		ProjectionExpression:     expr.Projection(),
		ExpressionAttributeNames: expr.Names(),
	}

	paginator := dynamodb.NewScanPaginator(d.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}

		for _, item := range output.Items {
			if jobId, exists := item["JobId"]; exists {
				if jobIdStr := jobId.(*types.AttributeValueMemberS); jobIdStr != nil {
					jobIds[jobIdStr.Value] = true
				}
			}
		}
	}
	utils.Debug(fmt.Sprintf("📊 Retrieved %d job IDs from DynamoDB", len(jobIds)))
	return jobIds, nil
}

func (d *dynamoDBClientImpl) WriteJobIdsToFile(filename string, keySet map[string]bool) error {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("failed to create file %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for key := range keySet {
		_, err := writer.WriteString(string(key) + "\n")
		if err != nil {
			fmt.Printf("failed to write key to file %v", err)
		}
	}
	err = writer.Flush()
	if err != nil {
		fmt.Printf("failed to flush writer: %v\n", err)
	}
	return nil
}
