package crawl

import (
	"context"
	"fmt"
	"go-crawler/internal/model"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Engine struct {
	workerCount int
	client      *http.Client
}

func NewEngine(workerCount int) *Engine {
	return &Engine{
		workerCount: workerCount,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
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
	// TODO Lesson 2: HTTP fetch

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
	// TODO Lesson 3: Extract links, enqueue with activeCount.Add(int32(len(links)))
	// TODO Lesson 4: Check MaxDepth, MaxPages
	// TODO Lesson 5: Save Page to PageStore
	_ = task
	_ = job
	_ = ctx
}
