output "lambda_function_name" {
  description = "The name of the Lambda function created."
  value       = aws_lambda_function.vapor_client_lambda.function_name
}

output "lambda_function_arn" {
  description = "The ARN of the Lambda function created."
  value       = aws_lambda_function.vapor_client_lambda.arn
}

output "lambda_iam_role_name" {
  description = "The name of the IAM role created for the Lambda function."
  value       = aws_iam_role.lambda_exec_role.name
}

output "lambda_iam_role_arn" {
  description = "ARN of the IAM role created for the Lambda function"
  value       = aws_iam_role.lambda_exec_role.arn
}

output "api_endpoint" {
  description = "URL of the API Gateway endpoint for the Lambda"
  value       = aws_apigatewayv2_api.http_api.api_endpoint
}

output "ecr_repository_url" {
  description = "URL of the ECR repository for vapor-server"
  value       = aws_ecr_repository.vapor_server_repo.repository_url
}

output "vapor_server_url" {
  description = "Public URL for the Vapor Server application"
  value       = "http://${aws_lb.vapor_server_lb.dns_name}"
}

output "db_instance_endpoint" {
  description = "Endpoint address for the RDS database instance"
  value       = aws_db_instance.vapor_db.address
  sensitive   = true
}

output "db_instance_port" {
  description = "Port for the RDS database instance"
  value       = aws_db_instance.vapor_db.port
}

output "api_gateway_invoke_url" {
  description = "The invoke URL for the HTTP API Gateway"
  value       = aws_apigatewayv2_api.http_api.api_endpoint
}
