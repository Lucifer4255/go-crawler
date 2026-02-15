package service

import (
	"context"
	"go-crawler/internal/model"
	"go-crawler/internal/store"
	"time"

	"github.com/google/uuid"
)

// JobRepository defines job persistence used by the service.
type JobRepository interface {
	CreateJob(job *model.CrawlJob) error
	GetJob(id string) (*model.CrawlJob, error)
	UpdateJobStatus(id string, status model.CrawlStatus, errMsg string) error
	TryIncrementPagesCrawled(id string, max int) (bool, error)
}

// PageRepository defines page persistence used by the service.
type PageRepository interface {
	CreatePage(page *model.Page) error
	GetPagesByJobID(jobID string) ([]*model.Page, error)
}

// CrawlRunner runs a single crawl job. Implemented by the crawl engine.
type CrawlRunner interface {
	Start(ctx context.Context, job *model.CrawlJob) error
}

// jobStoreAdapter adapts *store.JobStore to JobRepository (CreateJob returns error for interface compatibility).
type jobStoreAdapter struct {
	store *store.JobStore
}

// NewJobStoreAdapter returns a JobRepository that delegates to the given JobStore.
func NewJobStoreAdapter(s *store.JobStore) JobRepository {
	return &jobStoreAdapter{store: s}
}

func (a *jobStoreAdapter) CreateJob(job *model.CrawlJob) error {
	a.store.CreateJob(job)
	return nil
}

func (a *jobStoreAdapter) GetJob(id string) (*model.CrawlJob, error) {
	return a.store.GetJob(id)
}

func (a *jobStoreAdapter) UpdateJobStatus(id string, status model.CrawlStatus, errMsg string) error {
	return a.store.UpdateJobStatus(id, status, errMsg)
}

func (a *jobStoreAdapter) TryIncrementPagesCrawled(id string, max int) (bool, error) {
	return a.store.TryIncrementPagesCrawled(id, max)
}

// pageStoreAdapter adapts *store.PageStore to PageRepository (CreatePage returns error for interface compatibility).
type pageStoreAdapter struct {
	store *store.PageStore
}

// NewPageStoreAdapter returns a PageRepository that delegates to the given PageStore.
func NewPageStoreAdapter(s *store.PageStore) PageRepository {
	return &pageStoreAdapter{store: s}
}

func (a *pageStoreAdapter) CreatePage(page *model.Page) error {
	a.store.CreatePage(page)
	return nil
}

func (a *pageStoreAdapter) GetPagesByJobID(jobID string) ([]*model.Page, error) {
	return a.store.GetPagesByJobID(jobID)
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
	if err := s.jobs.CreateJob(job); err != nil {
		return nil, err
	}
	if err := s.jobs.UpdateJobStatus(job.ID, model.CrawlStatusRunning, ""); err != nil {
		return nil, err
	}
	go func() {
		err := s.runner.Start(ctx, job)
		if err != nil {
			_ = s.jobs.UpdateJobStatus(job.ID, model.CrawlStatusFailed, err.Error())
		} else {
			_ = s.jobs.UpdateJobStatus(job.ID, model.CrawlStatusCompleted, "")
		}
	}()
	return job, nil
}

// GetJob returns a job by ID.
func (s *CrawlService) GetJob(id string) (*model.CrawlJob, error) {
	return s.jobs.GetJob(id)
}

// GetPagesByJobID returns all pages stored for the given job.
func (s *CrawlService) GetPagesByJobID(jobID string) ([]*model.Page, error) {
	return s.pages.GetPagesByJobID(jobID)
}
