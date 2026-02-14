package service

import (
	"go-crawler/internal/model"

	"time"

	"github.com/google/uuid"
)

// /interfaces defined
type JobRepository interface {
	CreateJob(job *model.CrawlJob) error
	GetJob(id string) (*model.CrawlJob, error)
	UpdateJobStatus(id string, status model.CrawlStatus, errMsg string) error
	IncrementPagesCrawled(id string) error
}

///crawlservice - orchestration

type CrawlService struct {
	jobs JobRepository
}

// constructor
func NewCrawlService(jobs JobRepository) *CrawlService {
	return &CrawlService{
		jobs: jobs,
	}
}

// startcrawl - orchestration
func (s *CrawlService) Submit(input model.CrawlInput) (*model.CrawlJob, error) {
	job := &model.CrawlJob{
		ID:        uuid.New().String(),
		Input:     input,
		Status:    model.CrawlStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.jobs.CreateJob(job)
	return job, nil
}

func (s *CrawlService) GetJob(id string) (*model.CrawlJob, error) {
	return s.jobs.GetJob(id)
}
