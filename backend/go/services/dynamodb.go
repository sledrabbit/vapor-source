package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"gopher-source/models"
	"gopher-source/utils"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBClient interface {
	CreateJobsTable(ctx context.Context) error
	PutJob(ctx context.Context, job *models.Job) error
	GetAllJobIds(ctx context.Context) (map[string]bool, error)
	WriteJobIdsToFile(filename string, keySet map[string]bool) error
}

type dynamoDBClientImpl struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoService(cfg aws.Config, tableName string) DynamoDBClient {
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:8000")
	})
	return &dynamoDBClientImpl{client: client, tableName: tableName}
}

func NewDynamoConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-west-2"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	return cfg, nil
}

func (d *dynamoDBClientImpl) CreateJobsTable(ctx context.Context) error {
	// var tableDesc *types.TableDescription
	_, err := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("JobId"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("PostedDate"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("PostedDate-Index"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("PostedDate"),
						KeyType:       types.KeyTypeHash,
					},
					{
						AttributeName: aws.String("JobId"),
						KeyType:       types.KeyTypeRange,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeAll,
				},
			},
		},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String("JobId"),
			KeyType:       types.KeyTypeHash,
		}},
		TableName:   aws.String(d.tableName),
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		log.Printf("Couldn't create table %v. Here's why: %v\n", d.tableName, err)
	} else {
		waiter := dynamodb.NewTableExistsWaiter(d.client)
		err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(d.tableName)}, 5*time.Minute)
		if err != nil {
			log.Printf("Wait for table exists failed. Here's why: %v\n", err)
		}
		// tableDesc = table.TableDescription
		log.Printf("Creating table %s", d.tableName)
	}
	return nil
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
	utils.Debug("\t📦 Post successful: for job")
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
