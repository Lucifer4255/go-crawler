package crawl

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// PagesCrawledLimiter is used by the engine to check and increment page count under a limit.
// Implemented by the service layer (e.g. adapter over JobStore).
type PagesCrawledLimiter interface {
	TryIncrementPagesCrawled(jobID string, maxPages int) (allowed bool, err error)
}

// PageWriter is used by the engine to persist crawled pages.
// Implemented by the service layer (e.g. adapter over PageStore).
type PageWriter interface {
	CreatePage(page *model.Page) error
}

type Engine struct {
	workerCount  int
	client       *http.Client
	allowedHost  string
	pagesLimiter PagesCrawledLimiter
	pageWriter   PageWriter
}

func NewEngine(workerCount int, pagesLimiter PagesCrawledLimiter, pageWriter PageWriter) *Engine {
	return &Engine{
		workerCount: workerCount,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		pagesLimiter: pagesLimiter,
		pageWriter:   pageWriter,
	}
}

func (e *Engine) Start(ctx context.Context, job *model.CrawlJob) error {
	urlQueue := make(chan *model.URLTask, 1000)
	var wg sync.WaitGroup

	visitedURL := NewVisitedURLStore()

	// Use an atomic counter to track active tasks (queued + in-progress).
	// Bump up on enqueue, down on finish. When it hits zero, close the queue.
	// Only one goroutine should close.
	var activeCount atomic.Int32
	activeCount.Store(1) // Seed task we're about to enqueue

	seedUrl, err := url.Parse(job.Input.StartURL)
	if err != nil {
		return fmt.Errorf("invalid start URL: %w", err)
	}
	e.allowedHost = seedUrl.Host
	// Seed initial URL
	urlQueue <- &model.URLTask{
		URL:   job.Input.StartURL,
		Depth: 0,
	}

	for i := 0; i < e.workerCount; i++ {
		wg.Add(1)
		go e.worker(ctx, &wg, urlQueue, visitedURL, job, &activeCount)
	}
	wg.Wait()
	// Channel is closed by the worker that decrements activeCount to 0
	return nil
}

func (e *Engine) worker(ctx context.Context, wg *sync.WaitGroup, urlQueue chan *model.URLTask, visitedURL *VisitedURLStore, job *model.CrawlJob, activeCount *atomic.Int32) {
	defer wg.Done()
	for task := range urlQueue {
		e.processTask(ctx, urlQueue, visitedURL, job, task, activeCount)
	}
}

// processTask handles one URL. When done, decrements activeCount. If it
// enqueues child tasks, it increments activeCount for each.
func (e *Engine) processTask(ctx context.Context, urlQueue chan *model.URLTask, visitedURL *VisitedURLStore, job *model.CrawlJob, task *model.URLTask, activeCount *atomic.Int32) {
	defer func() {
		// Decrement: we're done with this task (processed or skipped).
		// The worker that brings activeCount to 0 closes the channel.
		if activeCount.Add(-1) == 0 {
			close(urlQueue)
		}
	}()

	if ctx.Err() != nil {
		return
	}
	// Skip if already visited (dedupe)
	visited := visitedURL.MarkIfNotVisited(task.URL)
	if !visited {
		fmt.Println("Skipping already visited URL:", task.URL)
		return
	}

	fmt.Println("Fetching:", task.URL)
	// -------------------------HTTP FETCH --------------------------

	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	// Many sites (golang.org, go.dev, google.com) return 403 or redirect for default "Go-http-client/1.1"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := e.client.Do(req)
	if err != nil {
		fmt.Println("[crawl] Error fetching URL:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("[crawl] Response status:", resp.StatusCode, "for", task.URL)
	//only continue if the response is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Println("[crawl] Skipping: not OK", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println("Fetched:", task.URL, "with status:", resp.StatusCode, "and body length:", len(body))

	// -------------------------MAX PAGES CHECK --------------------------

	allowed, err := e.pagesLimiter.TryIncrementPagesCrawled(job.ID, job.Input.MaxPages)
	if err != nil {
		fmt.Println("[crawl] Error incrementing pages crawled:", err)
		return
	}
	if !allowed {
		fmt.Println("[crawl] Max pages reached:", job.Input.MaxPages)
		return
	}

	// -------------------------PARSE PAGE --------------------------
	parsedPage, err := ParsePage(task.URL, body)
	if err != nil {
		fmt.Println("[crawl] Error parsing page:", err)
		return
	}

	// -------------------------SAVE PAGE --------------------------
	page := &model.Page{
		ID:           uuid.New().String(),
		JobID:        job.ID,
		URL:          task.URL,
		Title:        parsedPage.Title,
		Content:      string(body),
		DiscoveredAt: time.Now(),
	}
	if err := e.pageWriter.CreatePage(page); err != nil {
		fmt.Println("[crawl] Error saving page:", err)
		return
	}
	fmt.Println("[crawl] Saved page:", page.URL, "job:", job.ID)

	// -------------------------MAX DEPTH CHECK --------------------------

	if task.Depth >= job.Input.MaxDepth {
		fmt.Println("Max depth reached:", task.Depth)
		return
	}
	// -------------------------LINK EXTRACTION --------------------------

	fmt.Println("Extracted links:", parsedPage.Links)
	for _, link := range parsedPage.Links {
		if ctx.Err() != nil {
			return
		}

		childUrl, err := url.Parse(link)
		if err != nil {
			fmt.Println("Error parsing link:", err)
			continue
		}
		if childUrl.Host != e.allowedHost {
			fmt.Println("Skipping external link:", link)
			continue
		}
		visited := visitedURL.MarkIfNotVisited(link)
		if visited {
			fmt.Println("Enqueuing:", link)
			activeCount.Add(1)
			urlQueue <- &model.URLTask{
				URL:   link,
				Depth: task.Depth + 1,
			}
		}
	}

}
