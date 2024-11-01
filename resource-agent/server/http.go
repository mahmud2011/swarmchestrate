package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/mahmud2011/swarmchestrate/resource-agent/internal/config"
)

var (
	once     sync.Once
	instance *httpServer
)

type httpServer struct {
	s *http.Server
}

func NewHttpServer(conf config.HTTPServer, handler http.Handler) IServer {
	once.Do(func() {
		instance = &httpServer{
			s: &http.Server{
				Addr:    fmt.Sprintf("%s:%d", conf.Host, conf.Port),
				Handler: handler,
			},
		}
	})
	return instance
}

func (h *httpServer) Start() error {
	return h.s.ListenAndServe()
}

func (h *httpServer) Stop(ctx context.Context) error {
	return h.s.Shutdown(ctx)
}
