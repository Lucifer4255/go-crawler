package http

import (
	"go-crawler/internal/service"
	"net/http"
)

type Server struct {
	router  *http.ServeMux
	service *service.CrawlService
}

func NewServer(svc *service.CrawlService) *Server {
	server := &Server{
		router:  http.NewServeMux(),
		service: svc,
	}
	server.router.HandleFunc("/crawl", server.handleCrawl)
	server.router.HandleFunc("/crawl/{id}", server.handleGetJob)
	server.router.HandleFunc("/crawl/{id}/pages", server.handleGetPages)
	return server
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
