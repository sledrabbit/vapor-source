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

  global_secondary_index {
    name            = "PostedDate-Index"
    hash_key        = "PostedDate"
    range_key       = "JobId"
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
