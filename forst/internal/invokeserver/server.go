package invokeserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"forst/internal/discovery"
	"forst/internal/httpbody"
)

// marshalVersionPayload encodes VersionInfo for GET /version; tests may replace to inject errors.
var marshalVersionPayload = func(v VersionInfo) ([]byte, error) { return json.Marshal(v) }

// marshalFunctionList encodes the function list for GET /functions; tests may replace to inject errors.
var marshalFunctionList = func(list []discovery.FunctionInfo) ([]byte, error) { return json.Marshal(list) }

// Server is the shared HTTP invoke server for dev and embedded runtimes.
type Server struct {
	cfg        Config
	backend    DispatchBackend
	version    VersionInfo
	log        Logger
	server     *http.Server
	auth       *authState
	nonces     *nonceStore
	backoff    *failedAuthLimiter
	limiter    *concurrencyLimiter
	peerReader peerCredentialReader
	mu         sync.RWMutex
	started    bool
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
	if cfg.Transport == "" {
		if cfg.SocketPath != "" {
			cfg.Transport = transportUnix
		} else {
			cfg.Transport = transportTCP
		}
	}
	s := &Server{
		cfg:        cfg,
		backend:    backend,
		version:    version,
		log:        log,
		backoff:    newFailedAuthLimiter(),
		limiter:    newConcurrencyLimiter(cfg.MaxConcurrentInvoke),
		peerReader: defaultPeerCredentialReader(),
	}
	if cfg.authEnabled() {
		s.auth = newAuthState()
		if err := s.auth.initToken(); err != nil && log != nil {
			log.Errorf("invoke server: init auth: %v", err)
		}
		s.nonces = newNonceStore(30 * time.Second)
	}
	return s
}

// AuthEnabled reports whether invoke proof auth is active.
func (s *Server) AuthEnabled() bool {
	return s.authEnabled()
}

func (s *Server) authEnabled() bool {
	return s.auth != nil && s.nonces != nil
}

// CurrentAuth returns a copy of the live token and generation.
func (s *Server) CurrentAuth() (token []byte, generation uint64) {
	if s.auth == nil {
		return nil, 0
	}
	generation, token = s.auth.snapshot()
	return token, generation
}

// InstallAuth replaces the live auth secret (reload / handoff).
func (s *Server) InstallAuth(generation uint64, token []byte) {
	if s.auth == nil {
		s.auth = newAuthState()
	}
	if generation == 0 {
		s.auth.rotate(token)
		return
	}
	s.auth.mu.Lock()
	for i := range s.auth.token {
		s.auth.token[i] = 0
	}
	s.auth.token = append([]byte(nil), token...)
	s.auth.generation = generation
	s.auth.mu.Unlock()
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

// HandleChallenge handles GET /invoke/challenge.
func (s *Server) HandleChallenge(w http.ResponseWriter, r *http.Request) {
	s.handleChallenge(w, r)
}

// RegisterRoutes mounts invoke HTTP handlers on mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/version", s.handleVersion)
	mux.HandleFunc("/functions", s.handleFunctions)
	mux.HandleFunc("/invoke/challenge", s.handleChallenge)
	mux.HandleFunc("/invoke", s.handleInvoke)
}

// StartOnMux listens using an existing mux (caller may add extra routes first).
func (s *Server) StartOnMux(mux *http.ServeMux) error {
	if err := s.backend.RefreshFunctions(context.Background()); err != nil && s.log != nil {
		s.log.Errorf("invoke server: refresh functions on startup: %v", err)
	}
	s.RegisterRoutes(mux)

	ln, err := s.listen()
	if err != nil {
		return err
	}

	if err := s.afterListen(ln, mux); err != nil {
		_ = ln.Close()
		return err
	}
	return s.server.Serve(ln)
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

	ln, err := s.listen()
	if err != nil {
		return fmt.Errorf("invoke server: listen %s: %w", s.cfg.ListenTarget(), err)
	}

	if err := s.afterListen(ln, mux); err != nil {
		_ = ln.Close()
		return err
	}
	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed && s.log != nil {
			s.log.Errorf("invoke server stopped: %v", err)
		}
	}()
	return nil
}

// afterListen records the bound address, optionally writes auth artifacts, and builds the HTTP server.
func (s *Server) afterListen(ln net.Listener, mux *http.ServeMux) error {
	s.mu.Lock()
	s.server = s.buildHTTPServer(mux)
	s.server.Addr = ln.Addr().String()
	s.started = true
	if s.cfg.network() == transportTCP {
		if _, port, err := net.SplitHostPort(s.server.Addr); err == nil && port != "" {
			s.cfg.Port = port
		}
	}
	s.mu.Unlock()

	if s.log != nil {
		s.log.Infof("invoke HTTP server listening on %s (runtime=%s transport=%s)", s.server.Addr, s.cfg.Runtime, s.cfg.network())
	}
	if s.cfg.BoundaryRoot != "" {
		if err := s.WriteAuthArtifacts(s.cfg.BoundaryRoot, s.cfg); err != nil {
			return fmt.Errorf("invoke server: write auth artifacts: %w", err)
		}
	}
	return nil
}

func (s *Server) listen() (net.Listener, error) {
	if s.cfg.network() == transportUnix {
		if s.cfg.SocketPath == "" {
			return nil, fmt.Errorf("invoke server: unix transport requires socket path")
		}
		return listenUnixSocket(s.cfg.SocketPath)
	}
	return net.Listen(transportTCP, s.cfg.Addr())
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

// BoundAddr returns the actual listen address after StartAsync, else Config().Addr().
func (s *Server) BoundAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.server != nil && s.server.Addr != "" {
		return s.server.Addr
	}
	if s.cfg.network() == transportUnix && s.cfg.SocketPath != "" {
		return s.cfg.SocketPath
	}
	return s.cfg.Addr()
}

func (s *Server) effectiveTimeouts() (read, write time.Duration) {
	read = time.Duration(s.cfg.ReadTimeout) * time.Second
	write = time.Duration(s.cfg.WriteTimeout) * time.Second
	if read <= 0 {
		read = 30 * time.Second
	}
	if write <= 0 {
		write = 30 * time.Second
	}
	return read, write
}

func (s *Server) buildHTTPServer(mux *http.ServeMux) *http.Server {
	readTimeout, writeTimeout := s.effectiveTimeouts()
	handler := s.authMiddleware(s.loggingMiddleware(mux))
	return &http.Server{
		Addr:         s.cfg.ListenTarget(),
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, connContextKey{}, c)
		},
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp := Response{Success: true, Output: "Forst HTTP server is healthy"}
	if root, err := resolveBoundaryRoot(); err == nil {
		if reloading, generation := ReadReloadMarker(root); reloading {
			resp.Reloading = true
			resp.Generation = generation
		}
	}
	s.sendJSON(w, r, resp)
}

func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.authEnabled() || s.auth == nil || s.nonces == nil {
		s.sendError(w, r, "unauthorized", http.StatusUnauthorized)
		return
	}
	now := time.Now()
	nonce, expiresAt, err := s.nonces.issue(now)
	if err != nil {
		s.sendError(w, r, safeErrorMessage(err.Error()), http.StatusInternalServerError)
		return
	}
	generation := s.auth.currentGeneration()
	payload, err := json.Marshal(ChallengeResponse{
		Nonce:      nonce,
		ExpiresAt:  expiresAt.UTC().Format(time.RFC3339),
		Generation: generation,
	})
	if err != nil {
		s.sendError(w, r, "failed to marshal challenge", http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, r, Response{Success: true, Result: payload})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload, err := marshalVersionPayload(s.version)
	if err != nil {
		s.sendError(w, r, safeErrorMessage(fmt.Sprintf("failed to marshal version: %v", err)), http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, r, Response{Success: true, Result: payload})
}

func (s *Server) handleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.backend.RefreshFunctions(r.Context()); err != nil {
		s.sendError(w, r, safeErrorMessage(fmt.Sprintf("Failed to discover functions: %v", err)), http.StatusInternalServerError)
		return
	}
	functions := s.backend.Functions()
	list := make([]discovery.FunctionInfo, 0)
	for _, pkgFuncs := range functions {
		for _, fn := range pkgFuncs {
			list = append(list, fn)
		}
	}
	resultData, err := marshalFunctionList(list)
	if err != nil {
		s.sendError(w, r, safeErrorMessage(fmt.Sprintf("Failed to marshal functions: %v", err)), http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, r, Response{Success: true, Result: resultData})
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isAllowedInvokeHost(r.Host, s.cfg.AllowedHosts) {
		s.sendError(w, r, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !requireJSONContentType(r) {
		s.sendError(w, r, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	if s.rejectInvokeIfReloading(w, r) {
		return
	}
	release, err := s.limiter.Acquire(r.Context())
	if err != nil {
		s.sendError(w, r, "request cancelled", http.StatusServiceUnavailable)
		return
	}
	defer release()

	s.mu.RLock()
	maxBytes := s.cfg.MaxRequestSize
	s.mu.RUnlock()
	if maxBytes <= 0 {
		maxBytes = httpbody.DefaultMaxBytes
	}
	body, err := httpbody.ReadAll(r.Body, maxBytes)
	if err != nil {
		if httpbody.IsTooLarge(err) {
			s.sendError(w, r, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		s.sendError(w, r, safeErrorMessage(fmt.Sprintf("Failed to read request: %v", err)), http.StatusBadRequest)
		return
	}
	var req InvokeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.sendError(w, r, safeErrorMessage(fmt.Sprintf("Failed to decode request: %v", err)), http.StatusBadRequest)
		return
	}
	if s.log != nil {
		s.log.Debugf("call %s.%s streaming=%v", req.Package, req.Function, req.Streaming)
	}

	functions := s.backend.Functions()
	pkgFuncs, ok := functions[req.Package]
	if !ok {
		s.sendError(w, r, fmt.Sprintf("Package %s not found", req.Package), http.StatusNotFound)
		return
	}
	fn, ok := pkgFuncs[req.Function]
	if !ok {
		s.sendError(w, r, fmt.Sprintf("Function %s not found in package %s", req.Function, req.Package), http.StatusNotFound)
		return
	}
	if req.Streaming && !fn.SupportsStreaming {
		s.sendError(w, r, fmt.Sprintf("Function %s does not support streaming", req.Function), http.StatusBadRequest)
		return
	}

	if req.Streaming {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		flusher, ok := w.(http.Flusher)
		if !ok {
			s.sendError(w, r, "Streaming not supported by server", http.StatusInternalServerError)
			return
		}
		results, err := s.backend.InvokeStream(r.Context(), req.Package, req.Function, req.Args)
		if err != nil {
			s.sendError(w, r, safeErrorMessage(fmt.Sprintf("Streaming execution failed: %v", err)), http.StatusInternalServerError)
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
		s.sendError(w, r, safeErrorMessage(fmt.Sprintf("Function execution failed: %v", err)), http.StatusInternalServerError)
		return
	}
	s.sendJSON(w, r, Response{
		Success:    result.Success,
		Output:     result.Output,
		Error:      result.Error,
		ErrorValue: result.ErrorValue,
		Result:     result.Result,
	})
}

func (s *Server) sendJSON(w http.ResponseWriter, r *http.Request, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(ContractVersionHTTPHeader, s.version.ContractVersion)
	if s.cfg.CORS {
		if origin, ok := s.corsOriginFor(r); ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, "+HeaderInvokeProof+", "+HeaderInvokeGeneration+", "+HeaderInvokeNonce)
		}
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) corsOriginFor(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return "", false
	}
	if len(s.cfg.CORSAllowedOrigins) == 0 {
		return "", false
	}
	for _, allowed := range s.cfg.CORSAllowedOrigins {
		if origin == allowed {
			return origin, true
		}
	}
	return "", false
}

func (s *Server) sendError(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	s.sendJSON(w, r, Response{Success: false, Error: safeErrorMessage(errorMsg)})
}

func (s *Server) sendReloading(w http.ResponseWriter, r *http.Request, generation uint64) {
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusServiceUnavailable)
	s.sendJSON(w, r, Response{
		Success:    false,
		Error:      "reloading",
		Reloading:  true,
		Generation: generation,
	})
}

func (s *Server) rejectInvokeIfReloading(w http.ResponseWriter, r *http.Request) bool {
	root, err := resolveBoundaryRoot()
	if err != nil {
		return false
	}
	if reloading, gen := ReadReloadMarker(root); reloading {
		s.sendReloading(w, r, gen)
		return true
	}
	return false
}

// WriteAuthArtifacts writes invoke.ready metadata and optional invoke.token secret file.
func (s *Server) WriteAuthArtifacts(workDir string, cfg Config) error {
	if err := writeInvokeReady(workDir, cfg, s.authGeneration()); err != nil {
		return err
	}
	if !s.authEnabled() {
		return nil
	}
	token, _ := s.CurrentAuth()
	if len(token) == 0 {
		return nil
	}
	if w, ok := openAuthHandoffWriter(EnvInvokeAuthFD); ok {
		gen := s.authGeneration()
		return writeAuthHandoff(w, gen, token)
	}
	return writeTokenFile(invokeTokenPath(workDir), token)
}

func (s *Server) authGeneration() uint64 {
	if s.auth == nil {
		return 0
	}
	return s.auth.currentGeneration()
}

// RemoveAuthArtifacts deletes invoke.ready and invoke.token under workDir.
func RemoveAuthArtifacts(workDir string) error {
	if err := os.Remove(invokeReadyPath(workDir)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return removeTokenFile(invokeTokenPath(workDir))
}

func invokeReadyPath(workDir string) string {
	return filepath.Join(workDir, ".forst", "invoke.ready")
}
