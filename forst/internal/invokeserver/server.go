package invokeserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"forst/internal/discovery"
	"forst/internal/executor"
	"forst/internal/httpbody"
	"forst/internal/invokedispatch"
)

// DevBackend wraps the dev FunctionExecutor and discovery.
type DevBackend struct {
	exec       *executor.FunctionExecutor
	discoverer *discovery.Discoverer
	functions  map[string]map[string]discovery.FunctionInfo
	mu         sync.RWMutex
}

// NewDevBackend creates a backend that recompiles and go-runs on each invoke.
func NewDevBackend(exec *executor.FunctionExecutor, discoverer *discovery.Discoverer) *DevBackend {
	return &DevBackend{
		exec:       exec,
		discoverer: discoverer,
		functions:  make(map[string]map[string]discovery.FunctionInfo),
	}
}

// RefreshFunctions re-discovers public functions.
func (b *DevBackend) RefreshFunctions(_ context.Context) error {
	functions, err := b.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("discover functions: %w", err)
	}
	b.mu.Lock()
	b.functions = functions
	b.mu.Unlock()
	return nil
}

// Functions returns the cached function map.
func (b *DevBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make(map[string]map[string]discovery.FunctionInfo, len(b.functions))
	for pkg, fns := range b.functions {
		out[pkg] = make(map[string]discovery.FunctionInfo, len(fns))
		for name, info := range fns {
			out[pkg][name] = info
		}
	}
	return out
}

// Invoke runs a function via the dev executor.
func (b *DevBackend) Invoke(_ context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	result, err := b.exec.ExecuteFunction(pkg, fn, args)
	if err != nil {
		return nil, err
	}
	return &invokedispatch.InvokeResult{
		Success: result.Success,
		Output:  result.Output,
		Error:   result.Error,
		Result:  result.Result,
	}, nil
}

// InvokeStream runs a streaming function via the dev executor.
func (b *DevBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	ch, err := b.exec.ExecuteStreamingFunction(ctx, pkg, fn, args)
	if err != nil {
		return nil, err
	}
	out := make(chan invokedispatch.StreamChunk)
	go func() {
		defer close(out)
		for item := range ch {
			out <- invokedispatch.StreamChunk{
				Data:   item.Data,
				Status: item.Status,
				Error:  item.Error,
			}
		}
	}()
	return out, nil
}

// RegistryBackend adapts invokedispatch.Registry to DispatchBackend.
type RegistryBackend struct {
	registry *invokedispatch.Registry
}

// NewRegistryBackend wraps a registry.
func NewRegistryBackend(registry *invokedispatch.Registry) *RegistryBackend {
	return &RegistryBackend{registry: registry}
}

// RefreshFunctions is a no-op for embedded registry.
func (b *RegistryBackend) RefreshFunctions(context.Context) error {
	return nil
}

// Functions returns registered metadata.
func (b *RegistryBackend) Functions() map[string]map[string]discovery.FunctionInfo {
	return b.registry.Functions()
}

// Invoke calls an in-process handler.
func (b *RegistryBackend) Invoke(ctx context.Context, pkg, fn string, args json.RawMessage) (*invokedispatch.InvokeResult, error) {
	return b.registry.Invoke(ctx, pkg, fn, args)
}

// InvokeStream delegates to the registry.
func (b *RegistryBackend) InvokeStream(ctx context.Context, pkg, fn string, args json.RawMessage) (<-chan invokedispatch.StreamChunk, error) {
	return b.registry.InvokeStream(ctx, pkg, fn, args)
}

// Server is the shared HTTP invoke server for dev and embedded runtimes.
type Server struct {
	cfg      Config
	backend  DispatchBackend
	version  VersionInfo
	log      Logger
	server   *http.Server
	mu       sync.RWMutex
	started  bool
}

// Logger is the minimal logging surface for the invoke server.
type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
	Debugf(format string, args ...any)
}

// New creates an invoke HTTP server.
func New(cfg Config, backend DispatchBackend, version VersionInfo, log Logger) *Server {
	if version.ContractVersion == "" {
		version.ContractVersion = HTTPContractVersion
	}
	return &Server{
		cfg:     cfg,
		backend: backend,
		version: version,
		log:     log,
	}
}

// SetMaxRequestSize updates the invoke request body limit (tests).
func (s *Server) SetMaxRequestSize(n int64) {
	s.mu.Lock()
	s.cfg.MaxRequestSize = n
	s.mu.Unlock()
}
func (s *Server) SetBackend(backend DispatchBackend) {
	s.mu.Lock()
	s.backend = backend
	s.mu.Unlock()
}

// BackendFunctions returns function metadata from the active backend.
func (s *Server) BackendFunctions() map[string]map[string]discovery.FunctionInfo {
	s.mu.RLock()
	backend := s.backend
	s.mu.RUnlock()
	if backend == nil {
		return nil
	}
	return backend.Functions()
}

// HandleHealth handles GET /health.
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	s.handleHealth(w, r)
}

// HandleVersion handles GET /version.
func (s *Server) HandleVersion(w http.ResponseWriter, r *http.Request) {
	s.handleVersion(w, r)
}

// HandleFunctions handles GET /functions.
func (s *Server) HandleFunctions(w http.ResponseWriter, r *http.Request) {
	s.handleFunctions(w, r)
}

// HandleInvoke handles POST /invoke.
func (s *Server) HandleInvoke(w http.ResponseWriter, r *http.Request) {
	s.handleInvoke(w, r)
}

// RegisterRoutes mounts invoke HTTP handlers on mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/version", s.handleVersion)
	mux.HandleFunc("/functions", s.handleFunctions)
	mux.HandleFunc("/invoke", s.handleInvoke)
}

// StartOnMux listens using an existing mux (caller may add extra routes first).
func (s *Server) StartOnMux(mux *http.ServeMux) error {
	if err := s.backend.RefreshFunctions(context.Background()); err != nil && s.log != nil {
		s.log.Errorf("invoke server: refresh functions on startup: %v", err)
	}
	s.RegisterRoutes(mux)

	readTimeout := time.Duration(s.cfg.ReadTimeout) * time.Second
	writeTimeout := time.Duration(s.cfg.WriteTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}

	s.mu.Lock()
	s.server = &http.Server{
		Addr:         s.cfg.Addr(),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	s.started = true
	s.mu.Unlock()

	if s.log != nil {
		s.log.Infof("invoke HTTP server listening on %s (runtime=%s)", s.cfg.Addr(), s.cfg.Runtime)
	}
	return s.server.ListenAndServe()
}

// Start listens until the server stops. Blocks the caller.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	return s.StartOnMux(mux)
}

// StartAsync binds the listener synchronously, then serves in a background goroutine.
func (s *Server) StartAsync() error {
	if err := s.backend.RefreshFunctions(context.Background()); err != nil && s.log != nil {
		s.log.Errorf("invoke server: refresh functions on startup: %v", err)
	}

	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	readTimeout := time.Duration(s.cfg.ReadTimeout) * time.Second
	writeTimeout := time.Duration(s.cfg.WriteTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}

	ln, err := net.Listen("tcp", s.cfg.Addr())
	if err != nil {
		return fmt.Errorf("invoke server: listen %s: %w", s.cfg.Addr(), err)
	}

	s.mu.Lock()
	s.server = &http.Server{
		Addr:         s.cfg.Addr(),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	s.started = true
	s.mu.Unlock()

	if s.log != nil {
		s.log.Infof("invoke HTTP server listening on %s (runtime=%s)", s.cfg.Addr(), s.cfg.Runtime)
	}
	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed && s.log != nil {
			s.log.Errorf("invoke server stopped: %v", err)
		}
	}()
	return nil
}

// Stop closes the listener.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

// Config returns the server configuration.
func (s *Server) Config() Config {
	return s.cfg
}

// handleHealth handles GET /health.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.sendJSON(w, Response{Success: true, Output: "Forst HTTP server is healthy"})
}

// handleVersion handles GET /version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload, err := json.Marshal(s.version)
	if err != nil {
		s.sendError(w, fmt.Sprintf("failed to marshal version: %v", err), http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, Response{Success: true, Result: payload})
}

// handleFunctions handles GET /functions.
func (s *Server) handleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.backend.RefreshFunctions(r.Context()); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to discover functions: %v", err), http.StatusInternalServerError)
		return
	}
	functions := s.backend.Functions()
	list := make([]discovery.FunctionInfo, 0)
	for _, pkgFuncs := range functions {
		for _, fn := range pkgFuncs {
			list = append(list, fn)
		}
	}
	resultData, err := json.Marshal(list)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to marshal functions: %v", err), http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, Response{Success: true, Result: resultData})
}

// handleInvoke handles POST /invoke.
func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	maxBytes := s.cfg.MaxRequestSize
	s.mu.RUnlock()
	if maxBytes <= 0 {
		maxBytes = httpbody.DefaultMaxBytes
	}
	body, err := httpbody.ReadAll(r.Body, maxBytes)
	if err != nil {
		if httpbody.IsTooLarge(err) {
			s.sendError(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		s.sendError(w, fmt.Sprintf("Failed to read request: %v", err), http.StatusBadRequest)
		return
	}
	var req InvokeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	functions := s.backend.Functions()
	pkgFuncs, ok := functions[req.Package]
	if !ok {
		s.sendError(w, fmt.Sprintf("Package %s not found", req.Package), http.StatusNotFound)
		return
	}
	fn, ok := pkgFuncs[req.Function]
	if !ok {
		s.sendError(w, fmt.Sprintf("Function %s not found in package %s", req.Function, req.Package), http.StatusNotFound)
		return
	}
	if req.Streaming && !fn.SupportsStreaming {
		s.sendError(w, fmt.Sprintf("Function %s does not support streaming", req.Function), http.StatusBadRequest)
		return
	}

	if req.Streaming {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.sendError(w, "Streaming not supported by server", http.StatusInternalServerError)
			return
		}
		results, err := s.backend.InvokeStream(r.Context(), req.Package, req.Function, req.Args)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Streaming execution failed: %v", err), http.StatusInternalServerError)
			return
		}
		encoder := json.NewEncoder(w)
		for result := range results {
			if err := encoder.Encode(result); err != nil {
				if s.log != nil {
					s.log.Errorf("encode streaming result: %v", err)
				}
				return
			}
			flusher.Flush()
		}
		return
	}

	result, err := s.backend.Invoke(r.Context(), req.Package, req.Function, req.Args)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Function execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, Response{
		Success: result.Success,
		Output:  result.Output,
		Error:   result.Error,
		Result:  result.Result,
	})
}

func (s *Server) sendJSON(w http.ResponseWriter, response Response) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg.CORS {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) sendError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	s.sendJSON(w, Response{Success: false, Error: errorMsg})
}
