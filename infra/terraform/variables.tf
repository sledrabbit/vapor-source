variable "aws_region" {
  description = "The AWS region to deploy resources in."
  type        = string
  default     = "us-west-2"
}

variable "function_name" {
  description = "The name of the Lambda function."
  type        = string
  default     = "vapor-client-lambda"
}

variable "api_name" {
  description = "Name for the API Gateway HTTP API"
  type        = string
  default     = "VaporClientApiTF"
}

variable "lambda_zip_path" {
  description = "The path to the Lambda deployment package zip file."
  type        = string
  default     = "../../backend/swift/vapor-client/vapor-client-lambda.zip"
}

variable "lambda_memory_size" {
  description = "The amount of memory allocated to the Lambda function (in MB)."
  type        = number
  default     = 128
}

variable "lambda_timeout" {
  description = "The maximum execution time for the Lambda function (in seconds)."
  type        = number
  default     = 300
}

variable "lambda_env_vars" {
  description = "A map of additional environment variables to pass to the Lambda function."
  type        = map(string)
  default     = {}
}

variable "openai_api_key" {
  description = "The API key for OpenAI."
  type        = string
  sensitive   = true
}

variable "api_server_url" {
  description = "The URL for the backend API server."
  type        = string
}

variable "job_query" {
  description = "The default job query string."
  type        = string
  default     = "software engineer"
}

variable "debug_output" {
  description = "Enable debug output logging."
  type        = bool
  default     = false
}

variable "api_dry_run" {
  description = "Enable API dry run mode (prevents actual API calls)."
  type        = bool
  default     = false
}

variable "prompt_path" {
  description = "Path to the LLM prompt file within the Lambda environment."
  type        = string
  default     = "/var/task/prompt.txt"
}

variable "openai_base_url" {
  description = "The base URL for the OpenAI API."
  type        = string
  default     = "https://api.openai.com/v1/chat/completions"
}

variable "openai_model" {
  description = "The OpenAI model to use."
  type        = string
  default     = "gpt-4.1-nano"
}

variable "scraper_max_pages" {
  description = "Maximum number of pages the scraper should process."
  type        = number
  default     = 2
}

variable "scraper_base_url" {
  description = "The base URL for the scraper target website."
  type        = string
  default     = "https://www.worksourcewa.com/"
}

variable "scraper_request_delay" {
  description = "Delay in seconds between scraper requests."
  type        = number
  default     = 1.0
}

variable "parser_max_concurrent_tasks" {
  description = "Maximum number of concurrent parsing tasks."
  type        = number
  default     = 5
}

variable "db_name" {
  description = "Name for the RDS PostgreSQL database"
  type        = string
  default     = "vapordb"
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

variable "db_master_username" {
  description = "Master username for the RDS database"
  type        = string
  default     = "vaporadmin"
  sensitive   = true
}

variable "db_instance_class" {
  description = "Instance class for the RDS database"
  type        = string
  default     = "db.t3.micro"
}

variable "ecr_repo_name" {
  description = "Name for the ECR repository for vapor-server"
  type        = string
  default     = "vapor-server"
}

variable "ecs_cluster_name" {
  description = "Name for the ECS cluster"
  type        = string
  default     = "vapor-cluster"
}

variable "ecs_service_name" {
  description = "Name for the ECS service for vapor-server"
  type        = string
  default     = "vapor-server-service"
}

variable "ecs_task_family" {
  description = "Family name for the ECS task definition"
  type        = string
  default     = "vapor-server-task"
}

variable "vapor_server_image_uri" {
  description = "Docker image URI for the vapor-server (e.g., <account_id>.dkr.ecr.<region>.amazonaws.com/vapor-server:latest)"
  type        = string
}

variable "vapor_server_container_port" {
  description = "Port the vapor-server container listens on"
  type        = number
  default     = 8080
}

variable "ecs_task_cpu" {
  description = "CPU units for the ECS task"
  type        = number
  default     = 256
}

variable "ecs_task_memory" {
  description = "Memory (in MiB) for the ECS task"
  type        = number
  default     = 512
}
