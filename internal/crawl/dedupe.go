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

func (s *VisitedURLStore) MarkAsVisited(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[url] = true
}

func (s *VisitedURLStore) IsVisited(url string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.urls[url]
}
