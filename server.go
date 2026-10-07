package todo

import (
	"context"
	"net/http"
	"time"
)

type Server struct{ httpServer *http.Server }

func NewServer(port string, handler http.Handler) *Server {
	return &Server{httpServer: &http.Server{
		Addr: ":" + port, Handler: handler, MaxHeaderBytes: 1 << 20,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
	}}
}

func (s *Server) Run() error { return s.httpServer.ListenAndServe() }

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		_ = s.httpServer.Close()
		return err
	}
	return nil
}
