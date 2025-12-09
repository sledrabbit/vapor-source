package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"gopher-source/models"
)

type S3Client interface {
	UploadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error
	DownloadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error
	WriteJobsToJSONLFile(filename string, jobs []models.Job) error
}

type s3ClientImpl struct {
	client *s3.Client
}

func NewS3Service(cfg aws.Config, optFns ...func(*s3.Options)) S3Client {
	client := s3.NewFromConfig(cfg, optFns...)
	return &s3ClientImpl{client: client}
}

func (s *s3ClientImpl) UploadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
	} else {
		defer file.Close()
		input := &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
			Body:   file,
		}
		if ct := contentTypeForKey(objectKey); ct != "" {
			input.ContentType = aws.String(ct)
		}
		_, err = s.client.PutObject(ctx, input)
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "EntityTooLarge" {
				log.Printf("Error while uploading object to %s. The object is too large.\n"+
					"To upload objects larger than 5GB, use the S3 console (160GB max)\n"+
					"or the multipart upload API (5TB max).", bucketName)
			} else {
				log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
					fileName, bucketName, objectKey, err)
			}
		} else {
			err = s3.NewObjectExistsWaiter(s.client).Wait(
				ctx, &s3.HeadObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)}, time.Minute)
			if err != nil {
				log.Printf("Failed attempt to wait for object %s to exist.\n", objectKey)
			}
		}
	}
	return err
}

func (s *s3ClientImpl) DownloadFile(ctx context.Context, bucketName string, objectKey string, fileName string) error {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			log.Printf("Can't get object %s from bucket %s. No such key exists.\n", objectKey, bucketName)
			err = noKey
		} else {
			log.Printf("Couldn't get object %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		}
		return err
	}
	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Couldn't create file %v. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. Here's why: %v\n", objectKey, err)
	}
	_, err = file.Write(body)
	return err
}

func (s *s3ClientImpl) WriteJobsToJSONLFile(filename string, jobs []models.Job) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create jsonl file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, job := range jobs {
		b, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("marshal job %s: %w", job.JobId, err)
		}
		if _, err := writer.Write(b); err != nil {
			return fmt.Errorf("write job: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush writer: %w", err)
	}
	return nil
}

func contentTypeForKey(key string) string {
	lower := strings.ToLower(strings.TrimSpace(key))
	switch {
	case strings.HasSuffix(lower, ".json"),
		strings.HasSuffix(lower, ".jsonl"),
		strings.HasSuffix(lower, ".json.gz"),
		strings.HasSuffix(lower, ".jsonl.gz"):
		return "application/json"
	case strings.HasSuffix(lower, ".txt"):
		return "text/plain"
	default:
		return ""
	}
}
