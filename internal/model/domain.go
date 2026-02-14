package model

import "time"

type CrawlStatus string

const (
	CrawlStatusPending   CrawlStatus = "PENDING"
	CrawlStatusRunning   CrawlStatus = "RUNNING"
	CrawlStatusCompleted CrawlStatus = "COMPLETED"
	CrawlStatusCancelled CrawlStatus = "CANCELLED"
	CrawlStatusFailed    CrawlStatus = "FAILED"
)

type CrawlInput struct {
	StartURL       string
	MaxDepth       int
	MaxPages       int
	SameDomainOnly bool
	RequestDelayMs int
}

type CrawlJob struct {
	ID     string
	Input  CrawlInput
	Status CrawlStatus

	PagesCrawled int
	Error        error

	CreatedAt time.Time
	UpdatedAt time.Time
}

type URLTask struct {
	URL   string
	Depth int
}

type Page struct {
	ID           string
	JobID        string
	URL          string
	Title        string
	Content      string
	DiscoveredAt time.Time
}

type IndexEntry struct {
	Term   string
	PageID string
}
