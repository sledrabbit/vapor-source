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

resource "aws_s3_bucket" "snapshots" {
  bucket = var.snapshot_bucket_name
}

resource "aws_s3_bucket_public_access_block" "snapshots" {
  bucket                  = aws_s3_bucket.snapshots.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "snapshots" {
  bucket = aws_s3_bucket.snapshots.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_cloudfront_origin_access_identity" "snapshots" {
  comment = "Access identity for snapshot bucket"
}

resource "aws_cloudfront_distribution" "snapshots" {
  enabled             = true
  comment             = "Snapshot JSONL distribution"
  price_class         = "PriceClass_100"
  wait_for_deployment = true

  origin {
    domain_name = aws_s3_bucket.snapshots.bucket_regional_domain_name
    origin_id   = "snapshots-s3-origin"

    s3_origin_config {
      origin_access_identity = aws_cloudfront_origin_access_identity.snapshots.cloudfront_access_identity_path
    }
  }

  default_cache_behavior {
    target_origin_id       = "snapshots-s3-origin"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]

    forwarded_values {
      query_string = false

      cookies {
        forward = "none"
      }
    }

    min_ttl     = 0
    default_ttl = 3600
    max_ttl     = 86400
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

resource "aws_s3_bucket_policy" "snapshots_cloudfront" {
  bucket = aws_s3_bucket.snapshots.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = aws_cloudfront_origin_access_identity.snapshots.iam_arn
        }
        Action = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.snapshots.arn}/*"
      }
    ]
  })
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
  name               = "${var.scraper_lambda_function_name}-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
}

resource "aws_iam_role_policy" "job_scraper" {
  name = "${var.scraper_lambda_function_name}-inline"
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
      },
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:HeadObject"
        ]
        Resource = "${aws_s3_bucket.snapshots.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.snapshots.arn
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "job_scraper" {
  name              = "/aws/lambda/${var.scraper_lambda_function_name}"
  retention_in_days = 14
}

resource "aws_lambda_function" "job_scraper" {
  function_name = var.scraper_lambda_function_name
  description   = var.scraper_lambda_description
  role          = aws_iam_role.job_scraper.arn

  architectures    = ["arm64"]
  filename         = var.scraper_lambda_zip_path
  source_code_hash = filebase64sha256(var.scraper_lambda_zip_path)
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  timeout          = 900
  memory_size      = 128

  environment {
    variables = merge(
      var.scraper_environment_variables,
      {
        DYNAMODB_TABLE_NAME = aws_dynamodb_table.jobs.name
        JOB_IDS_BUCKET      = aws_s3_bucket.job_id_cache.bucket
        JOB_IDS_S3_KEY      = var.job_ids_s3_key
        SNAPSHOT_BUCKET     = aws_s3_bucket.snapshots.bucket
      }
    )
  }

  depends_on = [aws_cloudwatch_log_group.job_scraper]
}

resource "aws_cloudwatch_event_rule" "job_scraper_schedule" {
  name                = "${var.scraper_lambda_function_name}-schedule"
  description         = "Schedule for triggering the Go scraper Lambda."
  schedule_expression = var.scraper_schedule_expression
}

resource "aws_cloudwatch_event_target" "job_scraper_schedule" {
  rule      = aws_cloudwatch_event_rule.job_scraper_schedule.name
  target_id = "job-scraper-lambda"
  arn       = aws_lambda_function.job_scraper.arn
}

resource "aws_lambda_permission" "job_scraper_schedule" {
  statement_id  = "AllowExecutionFromEventBridgeScraper"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.job_scraper.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.job_scraper_schedule.arn
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

resource "aws_iam_role" "job_snapshot" {
  name               = "${var.snapshot_lambda_function_name}-role"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
}

resource "aws_iam_role_policy" "job_snapshot" {
  name = "${var.snapshot_lambda_function_name}-inline"
  role = aws_iam_role.job_snapshot.id

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
          "dynamodb:Query",
          "dynamodb:DescribeTable",
          "dynamodb:Scan"
        ]
        Resource = [
          aws_dynamodb_table.jobs.arn,
          "${aws_dynamodb_table.jobs.arn}/index/PostedDate-Index"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:HeadObject"
        ]
        Resource = "${aws_s3_bucket.snapshots.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.snapshots.arn
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "job_snapshot" {
  name              = "/aws/lambda/${var.snapshot_lambda_function_name}"
  retention_in_days = 14
}

resource "aws_lambda_function" "job_snapshot" {
  function_name = var.snapshot_lambda_function_name
  description   = var.snapshot_lambda_description
  role          = aws_iam_role.job_snapshot.arn

  architectures    = ["arm64"]
  filename         = var.snapshot_lambda_zip_path
  source_code_hash = filebase64sha256(var.snapshot_lambda_zip_path)
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  timeout          = 900
  memory_size      = 128

  environment {
    variables = merge(
      var.snapshot_environment_variables,
      {
        DYNAMODB_TABLE_NAME = aws_dynamodb_table.jobs.name
        SNAPSHOT_BUCKET     = aws_s3_bucket.snapshots.bucket
      }
    )
  }

  depends_on = [aws_cloudwatch_log_group.job_snapshot]
}

resource "aws_lambda_function_url" "job_snapshot" {
  function_name      = aws_lambda_function.job_snapshot.arn
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

output "snapshot_bucket_name" {
  description = "S3 bucket for JSONL snapshots generated by the snapshot Lambda."
  value       = aws_s3_bucket.snapshots.bucket
}

output "snapshot_lambda_function_arn" {
  description = "ARN of the snapshot Lambda function."
  value       = aws_lambda_function.job_snapshot.arn
}

output "snapshot_lambda_function_url" {
  description = "Function URL endpoint for triggering the snapshot Lambda."
  value       = aws_lambda_function_url.job_snapshot.function_url
}

output "snapshot_distribution_domain" {
  description = "CloudFront domain serving snapshot JSONL files."
  value       = aws_cloudfront_distribution.snapshots.domain_name
}
