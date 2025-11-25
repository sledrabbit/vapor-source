variable "aws_region" {
  description = "AWS region for the Lambda deployment."
  type        = string
  default     = "us-west-2"
}

variable "lambda_function_name" {
  description = "Name of the Lambda function to create."
  type        = string
  default     = "go-job-scraper"
}

variable "lambda_zip_path" {
  description = "Path to the built Lambda zip created by make -C backend/go zip-lambda."
  type        = string
  default     = "../../../backend/go/bin/lambda.zip"
}

variable "lambda_description" {
  description = "Description for the Lambda function."
  type        = string
  default     = "Go job scraping pipeline"
}

variable "environment_variables" {
  description = "Environment variables passed into the Lambda function."
  type        = map(string)
  default = {
    QUERY               = "software engineer"
    DEBUG_OUTPUT        = "true"
    API_DRY_RUN         = "false"
    USE_JOB_ID_FILE     = "false"
    OPENAI_API_KEY      = ""
    AWS_REGION          = "us-west-2"
    DYNAMODB_TABLE_NAME = "Jobs"
    DYNAMODB_ENDPOINT   = ""
  }
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
