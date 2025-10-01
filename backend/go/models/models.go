package models

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Job struct {
	ID                        uint     `json:"id,omitempty"`
	JobId                     string   `json:"jobId"`
	Title                     string   `json:"title"`
	Company                   string   `json:"company"`
	Location                  string   `json:"location"`
	Modality                  string   `json:"modality,omitempty"`
	PostedDate                string   `json:"postedDate"`
	ExpiresDate               string   `json:"expiresDate,omitempty"`
	Salary                    string   `json:"salary"`
	URL                       string   `json:"url"`
	MinYearsExperience        int      `json:"minYearsExperience,omitempty"`
	MinDegree                 string   `json:"minDegree,omitempty"`
	Domain                    string   `json:"domain,omitempty"`
	Description               string   `json:"description"`
	ParsedDescription         string   `json:"parsedDescription,omitempty"`
	S3Pointer                 string   `json:"s3Pointer,omitempty"`
	Languages                 []string `json:"languages,omitempty"`
	Technologies              []string `json:"technologies,omitempty"`
	IsSoftwareEngineerRelated bool     `json:"IsSoftwareEngineerRelated"`
}

func (j *Job) ToDynamoDBItem() (map[string]types.AttributeValue, error) {
	item, err := attributevalue.MarshalMap(j)
	if err != nil {
		panic(err)
	}
	return item, nil
}

// OpenAI Structured Outputs response schema
type OpenAIJobParsingResponse struct {
	ParsedDescription         string   `json:"ParsedDescription" jsonschema_description:"A concise summary of the job role and key responsibilities"`
	DeadlineDate              string   `json:"DeadlineDate" jsonschema_description:"Deadline or expiry date for the job posting. Use 'Ongoing until requisition is closed' if not specified"`
	MinDegree                 string   `json:"MinDegree" jsonschema:"enum=Bachelor's,enum=Master's,enum=Ph.D,enum=Unspecified"`
	MinYearsExperience        int      `json:"MinYearsExperience" jsonschema_description:"Minimum years of professional experience required. CRITICAL RULES: 1) If job title contains 'Senior' or 'Sr.' set to at least 4 years, 2) If job title contains 'Principal', 'Staff', 'Lead', or 'Director' set to at least 7 years, 3) If job title contains 'Mid-level' set to at least 2 years, 4) Otherwise extract specific years from description, 5) If no experience mentioned and no seniority keywords, set to 0,minimum=0,maximum=25"`
	Modality                  string   `json:"Modality" jsonschema:"enum=Remote,enum=Hybrid,enum=In-Office" jsonschema_description:"Work arrangement. Default to 'In-Office' if unclear"`
	Domain                    string   `json:"Domain" jsonschema:"enum=Backend,enum=Full-Stack,enum=AI/ML,enum=Data,enum=QA,enum=Front-End,enum=Security,enum=DevOps,enum=Mobile,enum=Site Reliability,enum=Networking,enum=Embedded Systems,enum=Gaming,enum=Financial,enum=Other" jsonschema_description:"Technical domain. If description focuses on server-side or microservices development, choose 'Backend'"`
	Languages                 []string `json:"Languages" jsonschema_description:"Programming languages mentioned in the job. Only include programming languages, not spoken languages like English or Spanish"`
	Technologies              []string `json:"Technologies" jsonschema_description:"Software tools, frameworks, databases, and technologies mentioned in the job"`
	IsSoftwareEngineerRelated bool     `json:"IsSoftwareEngineerRelated" jsonschema_description:"Whether the job is primarily related to software engineering. Set to true only for roles that primarily involve coding or deep technical system design (Software Engineer, Developer, Data Scientist, ML Engineer, DevOps Engineer, SRE, QA Engineer). Set to false for Project Manager, Product Manager, Designer, Sales Engineer, IT Support, etc."`
}

type JobStats struct {
	TotalJobs      int64
	ProcessedJobs  int64
	SuccessfulJobs int64
	FailedJobs     int64
	UnrelatedJobs  int64
}

func (s *JobStats) PrintSummary(executionTime time.Duration) {
	totalJobs := atomic.LoadInt64(&s.TotalJobs)
	processedJobs := atomic.LoadInt64(&s.ProcessedJobs)
	successfulJobs := atomic.LoadInt64(&s.SuccessfulJobs)

	fmt.Printf("\n📊 Job Processing Statistics:\n")
	fmt.Printf("   Total Jobs Scraped: %d\n", totalJobs)
	fmt.Printf("   Jobs Processed: %d\n", processedJobs)
	fmt.Printf("   Unrelated Jobs: %d\n", atomic.LoadInt64(&s.UnrelatedJobs))
	fmt.Printf("   Successfully Parsed by OpenAI: %d\n", successfulJobs)
	fmt.Printf("   Failed to Parse: %d\n", atomic.LoadInt64(&s.FailedJobs))

	if totalJobs > 0 {
		fmt.Printf("   Success Rate: %.1f%%\n", float64(successfulJobs)/float64(totalJobs)*100)
	}

	fmt.Printf("   Execution Time: %.2f seconds\n", executionTime.Seconds())
	if executionTime.Seconds() > 0 {
		fmt.Printf("   Jobs per Second: %.2f\n", float64(processedJobs)/executionTime.Seconds())
	}
}
