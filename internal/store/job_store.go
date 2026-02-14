package store

import (
	"errors"
	"go-crawler/internal/model"
	"sync"
	"time"
)

type JobStore struct {
	jobs map[string]*model.CrawlJob
	mu   sync.RWMutex
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*model.CrawlJob),
	}
}

func (s *JobStore) CreateJob(job *model.CrawlJob) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	s.jobs[job.ID] = job
}

func (s *JobStore) GetJob(id string) (*model.CrawlJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, errors.New("job not found")
	}
	return job, nil
}

func (s *JobStore) UpdateJobStatus(id string, status model.CrawlStatus, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return errors.New("job not found")
	}
	job.Status = status
	job.UpdatedAt = time.Now()
	job.Error = errors.New(errMsg)
	return nil
}

func (s *JobStore) IncrementPagesCrawled(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return errors.New("job not found")
	}
	job.PagesCrawled++
	job.UpdatedAt = time.Now()

	return nil
}
