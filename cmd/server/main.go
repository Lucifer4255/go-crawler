package main

import (
	"context"
	"log"
	"time"

	"go-crawler/internal/crawl"
	"go-crawler/internal/model"
	"go-crawler/internal/service"
	"go-crawler/internal/store"
)

type Server struct {
	service *service.CrawlService
}

func NewServer(service *service.CrawlService) *Server {
	return &Server{service: service}
}

func main() {
	jobStore := store.NewJobStore()
	pageStore := store.NewPageStore()

	jobAdapter := service.NewJobStoreAdapter(jobStore)
	pageAdapter := service.NewPageStoreAdapter(pageStore)

	engine := crawl.NewEngine(4, jobAdapter, pageAdapter)
	svc := service.NewCrawlService(jobAdapter, pageAdapter, engine)

	ctx := context.Background()
	input := model.CrawlInput{
		StartURL:       "https://golang.org",
		MaxDepth:       1,
		MaxPages:       5,
		SameDomainOnly: true,
		RequestDelayMs: 0,
	}

	job, err := svc.Submit(ctx, input)
	if err != nil {
		log.Fatalf("Submit: %v", err)
	}
	log.Printf("Crawl submitted, job ID: %s", job.ID)

	time.Sleep(15 * time.Second)

	job, err = svc.GetJob(job.ID)
	if err != nil {
		log.Fatalf("GetJob: %v", err)
	}
	log.Printf("Job status: %s, pages crawled: %d", job.Status, job.PagesCrawled)

	pages, err := svc.GetPagesByJobID(job.ID)
	if err != nil {
		log.Fatalf("GetPagesByJobID: %v", err)
	}
	log.Printf("Pages stored: %d", len(pages))
	for i, p := range pages {
		log.Printf("  [%d] %s - %s", i+1, p.URL, p.Title)
	}
}
