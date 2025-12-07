# Vapor Source: Job Aggregation & Analysis Platform

Vapor Source is a backend system for scraping, enriching, and exporting software job postings. The current implementation is a cost-minimized Go serverless pipeline; the original Swift + Vapor stack remains in `backend/swift/` as legacy. My main motivation for this project is job boards rarely expose reliable filters for years of experience or true remote eligibility.

## Core Functionality (Go, current)

* **Web scraping:** Go Lambda crawls `worksourcewa.com`.
* **AI-powered enrichment:** OpenAI structured outputs normalize modality, domain, degree, skills, and years of experience.
* **Canonical storage:** Jobs are deduped and stored in DynamoDB (`JobId` PK, `PostedDate` sort key, `PostedDate-Index` GSI).
* **Job ID cache:** In-memory dedupe set is seeded from the S3 `job-ids.txt` and persisted back, so runs remain idempotent across invocations.
* **Snapshot export:** Snapshot Lambda writes per-day JSONL files to S3 and refreshes `snapshot-manifest.json` for consumers (fronted by CloudFront).
* **Legacy (Swift/Vapor):** Kept for reference; no longer the canonical path.

## Architecture Overview (Go serverless)

```
EventBridge (cron)
      ↓
Scraper Lambda (Go)
      ↓
DynamoDB (JobId PK, PostedDate SK; GSI PostedDate-Index)
      ↕
S3 job ID cache (`job-ids.txt`) ↔ in-memory dedupe set
      ↓
Snapshot Lambda (Go)
      ↓
S3 (per-day JSONL + snapshot-manifest.json) → CloudFront
```

* Scraper Lambda handles crawling, OpenAI enrichment, S3-synced job ID cache seeding/persisting, dedupe, and writes to DynamoDB.
* When new rows land, the scraper invokes the snapshot Lambda to emit per-day JSONL files.
* CloudFront serves snapshot artifacts without exposing the S3 bucket.

## Technology Stack

* **Languages:** Go (current serverless), Swift (legacy)
* **AI & parsing:** OpenAI structured outputs
* **Data plane:** DynamoDB, S3, CloudFront
* **Compute:** AWS Lambda (+ EventBridge triggers)
* **IaC:** Terraform (`infra/terraform/go-serverless`)
* **Legacy:** Swift Vapor API + PostgreSQL (retained in `backend/swift/`)

## Project Structure

* `backend/go/`: Go Lambdas (`cmd/scraper`, `cmd/snapshot`, `cmd/local`) and shared libs.
* `backend/swift/`: Legacy Swift Lambda + Vapor server.
* `infra/terraform/go-serverless/`: Terraform for the Go stack (Lambdas, DynamoDB, S3, CloudFront, EventBridge).

## Getting Started (Go)

1. **Local run:** `cd backend/go && go run ./cmd/local` (requires `.env` with OpenAI key, AWS creds, query, etc.).
2. **Tests:** `cd backend/go && go test ./...`.
3. **Package Lambdas:** `cd backend/go && make zip-scraper && make zip-snapshot` → `bin/scraper/lambda.zip`, `bin/snapshot/lambda.zip`.
4. **Deploy (Terraform):** `cd infra/terraform/go-serverless && terraform init && terraform apply -var-file=terraform.tfvars`.

### Key configuration (Go)

* `OPENAI_API_KEY` (or `API_DRY_RUN=true`), `QUERY`, `MAX_PAGES`, `MAX_CONCURRENCY`.
* Cache flags: `USE_JOB_ID_FILE`, `USE_S3_JOB_ID_FILE`, `JOB_IDS_BUCKET`, `JOB_IDS_S3_KEY`.
* Data plane: `DYNAMODB_TABLE_NAME`, `SNAPSHOT_BUCKET`, `SNAPSHOT_S3_KEY`.
* Snapshot range overrides: `SNAPSHOT_START_DATE`, `SNAPSHOT_END_DATE`.
* `SNAPSHOT_LAMBDA_FUNCTION_NAME` so the scraper can trigger exports after new writes.

### Snapshot output

Per-day JSONL under `s3://<bucket>/<prefix>/<YYYY-MM-DD>.jsonl` plus `snapshot-manifest.json`. Example record:

```json
{"jobId":"abc123","title":"Senior Backend Engineer","postedDate":"2025-12-03","modality":"Remote","domain":"Backend","yearsExperience":5,"degree":"Bachelors","skills":["Go","AWS","DynamoDB"],"sourceUrl":"https://worksourcewa.com/..."}
```

### Limitations / TODO

* No frontend consumer yet; snapshot format is the contract for downstreams.
