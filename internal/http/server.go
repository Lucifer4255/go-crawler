package http

import (
	"go-crawler/internal/search"
	"go-crawler/internal/service"
	"net/http"
)

type Server struct {
	router  *http.ServeMux
	service *service.CrawlService
	index   *search.Index
}

func NewServer(svc *service.CrawlService, idx *search.Index) *Server {
	server := &Server{
		router:  http.NewServeMux(),
		service: svc,
		index:   idx,
	}
	server.router.HandleFunc("/crawl", server.handleCrawl)
	server.router.HandleFunc("/crawl/{id}", server.handleGetJob)
	server.router.HandleFunc("/crawl/{id}/pages", server.handleGetPages)
	server.router.HandleFunc("/search", server.handleSearch)
	return server
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
