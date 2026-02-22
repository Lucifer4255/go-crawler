package main

import (
	"context"
	"log"
	"os"

	"go-crawler/internal/crawl"
	httppkg "go-crawler/internal/http"
	"go-crawler/internal/repository"
	"go-crawler/internal/search"
	"go-crawler/internal/service"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // load .env if present; ignore error so prod can rely on real env
	ctx := context.Background()
	repo, err := repository.New(ctx, func() string {
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			log.Fatalf("DATABASE_URL is not set")
		}
		return dbURL
	}())
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close(ctx)
	index := search.NewIndex()
	pages, err := repo.ListPagesForIndex(ctx)
	if err != nil {
		log.Fatalf("Failed to list pages for index: %v", err)
	}
	index.BuildFromDocuments(pages)
	log.Println("Index built with", len(pages), "documents")
	pageRepositoryWriter := service.NewIndexingWriter(repo, index)
	engine := crawl.NewEngine(10, repo, pageRepositoryWriter)
	svc := service.NewCrawlService(repo, repo, engine)

	httpServer := httppkg.NewServer(svc, index, repo)
	log.Println("Starting server on port 8080")
	log.Fatal(httpServer.Start(":8080"))
}
