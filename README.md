# Vapor Source: Job Aggregation & Analysis Platform

Vapor Source is a backend cloud project (toy app) designed to scrape, process,
analyze, and store software job postings. It leverages Java and Go for its
backend services, OpenAI for intelligent parsing of job listings, and AWS for a
scalable cloud infrastructure managed by Terraform. Job postings in my
experience cannot be reliably filtered on years of experience required or if
they are actually remote, which is what inspired this project.

The original choice of Swift for the server and Lambda was simply to experience
it on the server side and check out the new language features. Currently porting
the server to Spring Boot to replace Swift Vapor server and the Lambda in Go for
serverless use.

## Core Functionality

- **Web Scraping:** Extracts job listings from `worksourcewa.com` using a Go
  scraper:
  - A **Go-based scraper** within an AWS Lambda function
    (`vapor-source/backend/go/services/scraper.go`) using the `Kanna` library
    for HTML parsing.
- **AI-Powered Job Data Enrichment:** The Go Lambda client utilizes the OpenAI
  API to parse raw job descriptions, extracting structured information such as:
  - Minimum degree and years of experience
  - Work modality (Remote, Hybrid, In-Office)
  - Job domain (Backend, Full-Stack, AI/ML, etc.)
  - Required programming languages and technologies
  - Relevance to software engineering roles
- **Centralized API & Data Storage:** A **Spring Boot Vapor server**
  (`vapor-source/backend/java/vapor-server`) provides a RESTful API (defined
  with OpenAPI) to:
  - Ingest processed job data from the Go Lambda client.
  - Persist job listings, associated languages, and technologies into a
    PostgreSQL database.
- **Automated Cloud Infrastructure:** The entire cloud deployment is managed via
  **Terraform** (`vapor-source/infra/terraform`), provisioning resources on AWS.

## Architecture Overview

1. **Scraping:**
   - The Go Lambda function (`gopher-source`) scrapes job data.
2. **Processing & Enrichment (Go Lambda):**
   - Job descriptions are sent to the OpenAI API for analysis and structuring.
3. **Ingestion & Storage (Vapor Server):**
   - The Go Lambda posts the enriched job data to the Vapor API.
   - The Vapor server validates and stores the data in an RDS PostgreSQL
     database.
4. **Deployment:**
   - The Go Lambda is packaged and deployed to AWS Lambda.
   - The Vapor server is containerized using Docker and deployed to AWS ECS
     Fargate.
   - An Application Load Balancer fronts the ECS service.
   - API Gateway exposes the Lambda function.
   - Database credentials for the Vapor server are securely managed using AWS
     SSM Parameter Store.

## Technology Stack

- **Languages:**
  - Java (Vapor API Server, Lambda Client/Scraper/Parser)
- **Backend Frameworks:**
  - Spring Boot
- **Web Scraping:**
  - Kanna (XML/HTML Parser)
- **AI & Data Processing:**
  - OpenAI API (GPT models for job description analysis)
- **Database:**
  - PostgreSQL
- **API & Client Generation:**
  - OpenAPI (for API design and client/server code generation)
  - Swagger UI
- **Cloud Platform (AWS):**
  - **Compute:** Lambda, ECS (Fargate)
  - **Database:** RDS for PostgreSQL
  - **Networking:** API Gateway, Application Load Balancer (ALB), VPC, Security
    Groups
  - **Containerization:** ECR (Elastic Container Registry), Docker
  - **Secrets Management:** SSM Parameter Store
  - **Logging:** CloudWatch Logs
- **Infrastructure as Code (IaC):**
  - Terraform
- **Build & Deployment Tools:**
  - AWS SAM (for local Lambda testing - `template.yaml` in `vapor-client`)
  - Docker & Docker Compose (for Vapor server development and deployment)

## Project Structure

- `vapor-source/backend/go/`: AWS Lambda function for scraping, AI parsing, and
  posting data.
- `vapor-source/backend/java/vapor-server/`: Vapor API server and database
  logic.
- `vapor-source/infra/terraform/`: Terraform configuration for AWS
  infrastructure.

## Getting Started

1. **Prerequisites:**
   - AWS Account & CLI configured
   - Terraform installed
   - Docker installed
2. **Infrastructure Deployment:**
   - Navigate to `vapor-source/infra/terraform/`.
   - Update `terraform.tfvars` with your specific configurations (e.g., OpenAI
     API key, ECR image URI after building and pushing the Vapor server image).
   - Initialize Terraform: `terraform init`
   - Plan and apply: `terraform plan` and `terraform apply`
3. **Application Deployment:**
   - **Vapor Server (ECS):**
     - Build the Docker image for the Vapor server
     - Push the image to the ECR repository created by Terraform.
     - Ensure the `vapor_server_image_uri` in `terraform.tfvars` points to this
       image. Terraform will then deploy it to ECS.
   - **Go Lambda Client:**
     - Ensure the `lambda_zip_path` in `terraform.tfvars` points to the
       generated zip file. Terraform will deploy the Lambda function.
