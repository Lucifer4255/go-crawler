package main

import (
	"log"

	"go-crawler/internal/crawl"
	httppkg "go-crawler/internal/http"
	"go-crawler/internal/service"
	"go-crawler/internal/store"
)

func main() {
	jobStore := store.NewJobStore()
	pageStore := store.NewPageStore()

	jobAdapter := service.NewJobStoreAdapter(jobStore)
	pageAdapter := service.NewPageStoreAdapter(pageStore)

	engine := crawl.NewEngine(4, jobAdapter, pageAdapter)
	svc := service.NewCrawlService(jobAdapter, pageAdapter, engine)

	httpServer := httppkg.NewServer(svc)
	log.Println("Starting server on port 8080")
	log.Fatal(httpServer.Start(":8080"))
}
