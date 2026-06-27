package devserver

import (
	"fmt"
	"net/http"
	"time"
)

// Start starts the HTTP server.
func (s *DevServer) Start() error {
	if err := s.discoverFunctions(); err != nil {
		s.log.Warnf("Failed to discover functions on startup: %v", err)
	}

	s.typesGenerator = NewTypeScriptGenerator(s.log)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/version", s.handleVersion)
	mux.HandleFunc("/functions", s.handleFunctions)
	mux.HandleFunc("/invoke", s.handleInvoke)
	mux.HandleFunc("/invoke/raw", s.handleInvokeRaw)
	mux.HandleFunc("/types", s.handleTypes)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  time.Duration(s.httpOpts.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(s.httpOpts.WriteTimeoutSec) * time.Second,
	}

	s.logStartupInfo()
	return s.server.ListenAndServe()
}

// logStartupInfo logs information about the server startup.
func (s *DevServer) logStartupInfo() {
	s.log.Infof("HTTP server listening on port %s", s.port)
	s.log.Info("Available endpoints:")
	s.log.Info("  GET  /functions  - Discover available functions")
	s.log.Info("  POST /invoke     - Invoke a Forst function")
	s.log.Info("  POST /invoke/raw - Invoke with raw JSON args array body (see HTTP contract)")
	s.log.Info("  GET  /types      - Generate TypeScript types for discovered functions")
	s.log.Info("  GET  /health     - Health check")
	s.log.Info("  GET  /version    - Compiler and HTTP contract version")
}

// Stop stops the HTTP server.
func (s *DevServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// discoverFunctions discovers all available functions.
func (s *DevServer) discoverFunctions() error {
	functions, err := s.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("failed to discover functions: %v", err)
	}

	s.mu.Lock()
	s.functions = functions
	s.mu.Unlock()

	return nil
}
