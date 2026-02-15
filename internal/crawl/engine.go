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
	workerCount   int
	client        *http.Client
	allowedHost   string
	pagesLimiter  PagesCrawledLimiter
	pageWriter    PageWriter
}

func NewEngine(workerCount int, pagesLimiter PagesCrawledLimiter, pageWriter PageWriter) *Engine {
	return &Engine{
		workerCount:  workerCount,
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

	// -----------------------------------------------------------------------
	// LESSON 1: Active Task Counter
	// -----------------------------------------------------------------------
	// Problem: Workers block on <-urlQueue. We must close the channel when
	// there's no more work. But WHO closes it? Only one goroutine can close.
	//
	// Solution: Track "active" tasks = in queue + being processed.
	// - activeCount: atomic int32
	// - Enqueue a task: activeCount++
	// - Worker finishes a task: activeCount--
	// - When activeCount hits 0: the LAST worker to decrement closes the channel
	//
	// Why atomic? Multiple workers read/write; mutex would work too, but
	// atomic is simpler for a single counter.
	// -----------------------------------------------------------------------
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

	resp, err := e.client.Do(req)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return
	}
	defer resp.Body.Close()

	//only continue if the response is OK
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Not OK response", resp.StatusCode)
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
		fmt.Println("Error incrementing pages crawled:", err)
		return
	}
	if !allowed {
		fmt.Println("Max pages reached:", job.Input.MaxPages)
		return
	}

	// -------------------------PARSE PAGE --------------------------
	parsedPage, err := ParsePage(task.URL, body)
	if err != nil {
		fmt.Println("Error parsing page:", err)
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
		fmt.Println("Error saving page:", err)
		return
	}

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
