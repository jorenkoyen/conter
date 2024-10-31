package server

import (
	"context"
	"net/http"

	"github.com/jorenkoyen/conter/manager"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
)

type Server struct {
	logger       *logger.Logger
	addr         string
	Orchestrator *manager.Container
	handler      http.Handler
}

// NewServer will create a new management HTTP server.
func NewServer(addr string) *Server {
	mux := NewMux()
	s := &Server{
		logger:  log.WithName("server"),
		addr:    addr,
		handler: mux,
	}

	// register middleware
	mux.Use(s.LoggerMiddleware())

	// register routes
	mux.Handle("POST /api/projects", s.HandleProjectApply)
	mux.Handle("GET /api/projects/{name}", s.HandleProjectRetrieve)
	mux.Handle("DELETE /api/projects/{name}", s.HandleProjectDelete)

	return s
}

// Listen will actively start listening for connections on the management HTTP address.
// It will gracefully shut down the HTTP server when the context is cancelled.
func (s *Server) Listen(ctx context.Context) error {
	server := &http.Server{Addr: s.addr, Handler: s.handler}

	go func() {
		<-ctx.Done()
		s.logger.Trace("Gracefully shutting down management server")
		if err := server.Shutdown(context.Background()); err != nil {
			s.logger.Errorf("Failed to shutdown server: %v", err)
		}
	}()

	// create HTTP server
	s.logger.Infof("Starting management server on address=%s", s.addr)
	return server.ListenAndServe()
}
