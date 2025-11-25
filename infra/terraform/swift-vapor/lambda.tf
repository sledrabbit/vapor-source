resource "aws_iam_role" "lambda_exec_role" {
  name = "${var.function_name}-exec-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })

  tags = {
    ManagedBy = "Terraform"
  }
}

# Policy to allow logging
resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.lambda_exec_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_lambda_function" "vapor_client_lambda" {
  filename      = var.lambda_zip_path
  function_name = var.function_name
  role          = aws_iam_role.lambda_exec_role.arn
  handler       = "Provided"

  source_code_hash = filebase64sha256(var.lambda_zip_path)

  runtime       = "provided.al2"
  architectures = ["arm64"]

  memory_size = var.lambda_memory_size
  timeout     = var.lambda_timeout

  environment {
    variables = merge(var.lambda_env_vars, {
      OPENAI_API_KEY              = var.openai_api_key
      API_SERVER_URL              = "http://${aws_lb.vapor_server_lb.dns_name}"
      QUERY                       = var.job_query
      DEBUG_OUTPUT                = var.debug_output
      API_DRY_RUN                 = var.api_dry_run
      LLM_PROMPT_PATH             = var.prompt_path
      OPENAI_BASE_URL             = var.openai_base_url
      OPENAI_MODEL                = var.openai_model
      SCRAPER_MAX_PAGES           = var.scraper_max_pages
      SCRAPER_BASE_URL            = var.scraper_base_url
      SCRAPER_REQUEST_DELAY       = var.scraper_request_delay
      PARSER_MAX_CONCURRENT_TASKS = var.parser_max_concurrent_tasks
    })
  }

  tags = {
    ManagedBy = "Terraform"
  }
}
