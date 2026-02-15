package crawl

import "sync"

type VisitedURLStore struct {
	urls map[string]bool
	mu   sync.RWMutex
}

func NewVisitedURLStore() *VisitedURLStore {
	return &VisitedURLStore{
		urls: make(map[string]bool),
	}
}

func (s *VisitedURLStore) MarkIfNotVisited(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.urls[url] {
		return false
	}
	s.urls[url] = true
	return true
}
