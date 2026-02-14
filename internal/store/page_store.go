package store

import (
	// "go-crawler/internal/model"

	"go-crawler/internal/model"
	"sync"
	"time"
)

type PageStore struct {
	pages map[string]*model.Page
	mu    sync.RWMutex
}

func NewPageStore() *PageStore {
	return &PageStore{
		pages: make(map[string]*model.Page),
	}
}

func (s *PageStore) CreatePage(page *model.Page) {
	s.mu.Lock()
	defer s.mu.Unlock()

	page.DiscoveredAt = time.Now()
	s.pages[page.ID] = page
}

func (s *PageStore) GetPagesByJobID(jobID string) ([]*model.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pages := make([]*model.Page, 0)
	for _, page := range s.pages {
		if page.JobID == jobID {
			pages = append(pages, page)
		}
	}
	return pages, nil
}
