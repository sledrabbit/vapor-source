terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.40"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

resource "aws_s3_bucket" "job_id_cache" {
  bucket = var.job_ids_bucket_name
}

resource "aws_s3_bucket_public_access_block" "job_id_cache" {
  bucket                  = aws_s3_bucket.job_id_cache.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "job_id_cache" {
  bucket = aws_s3_bucket.job_id_cache.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_dynamodb_table" "jobs" {
  name         = var.dynamodb_table_name
  billing_mode = var.dynamodb_billing_mode

  hash_key  = "JobId"
  range_key = "PostedDate"

  attribute {
    name = "JobId"
    type = "S"
  }

  attribute {
    name = "PostedDate"
    type = "S"
  }

  attribute {
    name = "PostedTime"
    type = "S"
  }

  global_secondary_index {
    name            = "PostedDate-Index"
    hash_key        = "PostedDate"
    range_key       = "PostedTime"
    projection_type = "ALL"
  }
}

data "aws_iam_policy_document" "lambda_assume" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "job_scraper" {
  name               = "${var.lambda_function_name}-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
}

resource "aws_iam_role_policy" "job_scraper" {
  name = "${var.lambda_function_name}-inline"
  role = aws_iam_role.job_scraper.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:BatchWriteItem",
          "dynamodb:Scan",
          "dynamodb:DescribeTable"
        ]
        Resource = aws_dynamodb_table.jobs.arn
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject"
        ]
        Resource = "${aws_s3_bucket.job_id_cache.arn}/${var.job_ids_s3_key}"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.job_id_cache.arn
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "job_scraper" {
  name              = "/aws/lambda/${var.lambda_function_name}"
  retention_in_days = 14
}

resource "aws_lambda_function" "job_scraper" {
  function_name = var.lambda_function_name
  description   = var.lambda_description
  role          = aws_iam_role.job_scraper.arn

  architectures    = ["arm64"]
  filename         = var.lambda_zip_path
  source_code_hash = filebase64sha256(var.lambda_zip_path)
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  timeout          = 900
  memory_size      = 128

  environment {
    variables = merge(
      var.environment_variables,
      {
        DYNAMODB_TABLE_NAME = aws_dynamodb_table.jobs.name
        JOB_IDS_BUCKET      = aws_s3_bucket.job_id_cache.bucket
        JOB_IDS_S3_KEY      = var.job_ids_s3_key
      }
    )
  }

  depends_on = [aws_cloudwatch_log_group.job_scraper]
}

resource "aws_lambda_function_url" "job_scraper" {
  function_name      = aws_lambda_function.job_scraper.arn
  authorization_type = "NONE"

  cors {
    allow_origins = ["*"]
    allow_methods = ["*"]
    allow_headers = ["*"]
  }
}

output "lambda_function_arn" {
  description = "ARN of the deployed Go job scraper Lambda."
  value       = aws_lambda_function.job_scraper.arn
}

output "dynamodb_table_name" {
  description = "Name of the DynamoDB table used by the Lambda."
  value       = aws_dynamodb_table.jobs.name
}

output "lambda_function_url" {
  description = "Public Function URL endpoint for the Lambda."
  value       = aws_lambda_function_url.job_scraper.function_url
}

output "job_ids_bucket_name" {
  description = "S3 bucket that stores the job ID cache file."
  value       = aws_s3_bucket.job_id_cache.bucket
}
