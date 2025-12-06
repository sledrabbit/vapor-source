variable "aws_region" {
  description = "AWS region for the Lambda deployment."
  type        = string
  default     = "us-west-2"
}

variable "scraper_lambda_function_name" {
  description = "Name of the scraping Lambda function."
  type        = string
  default     = "go-job-scraper"
}

variable "scraper_lambda_description" {
  description = "Description for the scraping Lambda function."
  type        = string
  default     = "Go job scraping pipeline"
}

variable "scraper_lambda_zip_path" {
  description = "Path to the built scraper Lambda zip created by make zip-scraper."
  type        = string
  default     = "../../../backend/go/bin/scraper/lambda.zip"
}

variable "snapshot_lambda_function_name" {
  description = "Name of the snapshot Lambda function."
  type        = string
  default     = "go-job-snapshot"
}

variable "snapshot_lambda_description" {
  description = "Description for the snapshot Lambda function."
  type        = string
  default     = "Dumps DynamoDB rows to S3 snapshots"
}

variable "snapshot_lambda_zip_path" {
  description = "Path to the built snapshot Lambda zip created by make zip-snapshot."
  type        = string
  default     = "../../../backend/go/bin/snapshot/lambda.zip"
}

variable "scraper_environment_variables" {
  description = "Environment variables passed into the scraper Lambda."
  type        = map(string)
  default = {
    QUERY               = "software engineer"
    DEBUG_OUTPUT        = "true"
    API_DRY_RUN         = "false"
    USE_JOB_ID_FILE     = "false"
    USE_S3_JOB_ID_FILE  = "false"
    OPENAI_API_KEY      = ""
    DYNAMODB_TABLE_NAME = "Jobs"
    DYNAMODB_ENDPOINT   = ""
    JOB_IDS_BUCKET      = ""
    JOB_IDS_S3_KEY      = ""
    SNAPSHOT_BUCKET     = ""
    SNAPSHOT_S3_KEY     = ""
    MAX_PAGES           = "5"
  }
}

variable "snapshot_environment_variables" {
  description = "Environment variables passed into the snapshot Lambda."
  type        = map(string)
  default = {
    DYNAMODB_TABLE_NAME = "Jobs"
    DYNAMODB_ENDPOINT   = ""
    SNAPSHOT_BUCKET     = ""
    SNAPSHOT_S3_KEY     = ""
    API_DRY_RUN         = "true"
    SNAPSHOT_START_DATE = ""
    SNAPSHOT_END_DATE   = ""
  }
}

variable "job_ids_bucket_name" {
  description = "Name of the S3 bucket that stores the job ID cache file."
  type        = string
}

variable "snapshot_bucket_name" {
  description = "Name of the S3 bucket where snapshot JSONL exports are stored."
  type        = string
}

variable "job_ids_s3_key" {
  description = "S3 object key for the job ID cache file."
  type        = string
  default     = "job-ids.txt"
}

variable "dynamodb_table_name" {
  description = "Name of the DynamoDB table backing the job cache."
  type        = string
  default     = "Jobs"
}

variable "dynamodb_billing_mode" {
  description = "Billing mode for the DynamoDB table."
  type        = string
  default     = "PAY_PER_REQUEST"
}
