package service

import (
	"context"
	"go-crawler/internal/model"
	"time"

	"github.com/google/uuid"
)

// JobRepository defines job persistence used by the service.
type JobRepository interface {
	CreateJob(ctx context.Context, job *model.CrawlJob) error
	GetJob(ctx context.Context, id string) (*model.CrawlJob, error)
	UpdateJobStatus(ctx context.Context, id string, status model.CrawlStatus, errMsg string) error
	TryIncrementPagesCrawled(ctx context.Context, id string, max int) (bool, error)
}

// PageRepository defines page persistence used by the service.
// UpsertPage persists a page (insert or update by URL) and returns the saved page with ID set.
type PageRepository interface {
	UpsertPage(ctx context.Context, page *model.Page) (*model.Page, error)
	GetPagesByJobID(ctx context.Context, jobID string) ([]*model.Page, error)
}

// PageRepositoryWriter extends PageRepository with CreatePage for use as crawl.PageWriter.
// The same implementation can be passed to CrawlService and to the engine.
type PageRepositoryWriter interface {
	PageRepository
	CreatePage(ctx context.Context, page *model.Page) error
}

// CrawlRunner runs a single crawl job. Implemented by the crawl engine.
type CrawlRunner interface {
	Start(ctx context.Context, job *model.CrawlJob) error
}

// CrawlService orchestrates crawl jobs and the engine.
type CrawlService struct {
	jobs   JobRepository
	pages  PageRepository
	runner CrawlRunner
}

// NewCrawlService builds a CrawlService with the given job repo, page repo, and crawl runner.
func NewCrawlService(jobs JobRepository, pages PageRepository, runner CrawlRunner) *CrawlService {
	return &CrawlService{
		jobs:   jobs,
		pages:  pages,
		runner: runner,
	}
}

// Submit creates a job, stores it, sets status to RUNNING, and starts the crawl in a goroutine.
// The job is returned immediately; status is updated to COMPLETED or FAILED when the crawl finishes.
func (s *CrawlService) Submit(ctx context.Context, input model.CrawlInput) (*model.CrawlJob, error) {
	job := &model.CrawlJob{
		ID:        uuid.New().String(),
		Input:     input,
		Status:    model.CrawlStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.jobs.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if err := s.jobs.UpdateJobStatus(ctx, job.ID, model.CrawlStatusRunning, ""); err != nil {
		return nil, err
	}
	// Run crawl with a background context so it continues after the HTTP response is sent.
	go func() {
		crawlCtx := context.Background()
		err := s.runner.Start(crawlCtx, job)
		if err != nil {
			_ = s.jobs.UpdateJobStatus(crawlCtx, job.ID, model.CrawlStatusFailed, err.Error())
		} else {
			_ = s.jobs.UpdateJobStatus(crawlCtx, job.ID, model.CrawlStatusCompleted, "")
		}
	}()
	return job, nil
}

// GetJob returns a job by ID.
func (s *CrawlService) GetJob(ctx context.Context, id string) (*model.CrawlJob, error) {
	return s.jobs.GetJob(ctx, id)
}

// GetPagesByJobID returns all pages stored for the given job.
func (s *CrawlService) GetPagesByJobID(ctx context.Context, jobID string) ([]*model.Page, error) {
	return s.pages.GetPagesByJobID(ctx, jobID)
}
