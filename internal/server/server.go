package server

import (
	"fmt"
	"net/http"
)

// Server is a simple development HTTP file server.
type Server struct {
	outputDir string
	port      int
}

// New returns a Server that serves files from outputDir on the given port.
func New(outputDir string, port int) *Server {
	return &Server{outputDir: outputDir, port: port}
}

// Start begins serving and blocks until the server stops.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	fs := http.FileServer(http.Dir(s.outputDir))
	if err := http.ListenAndServe(addr, fs); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
