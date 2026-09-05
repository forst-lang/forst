package invokeserver

import (
	"fmt"
	"sync"

	"forst/internal/ftconfig"
	"forst/internal/invokedispatch"
)

type embeddedDeps struct {
	resolveRoot func() (string, error)
	loadConfig  func(dir string) (*ftconfig.Config, error)
	newServer   func(cfg Config, backend DispatchBackend, version VersionInfo, log Logger) *Server
}

func defaultEmbeddedDeps() embeddedDeps {
	return embeddedDeps{
		resolveRoot: resolveBoundaryRoot,
		loadConfig:  ftconfig.LoadFromDir,
		newServer:   New,
	}
}

// EmbeddedRuntime manages an embedded invoke HTTP server instance.
type EmbeddedRuntime struct {
	registry   *invokedispatch.Registry
	server     *Server
	serverMu   sync.Mutex
	shutdown   chan struct{}
	shutdownMu sync.Mutex
	deps       embeddedDeps
	once       sync.Once
	startErr   error
}

func newEmbeddedRuntime(deps embeddedDeps) *EmbeddedRuntime {
	return &EmbeddedRuntime{deps: deps}
}

func (r *EmbeddedRuntime) registryOrNew() *invokedispatch.Registry {
	if r.registry == nil {
		r.registry = invokedispatch.NewRegistry()
	}
	return r.registry
}

// Start boots the embedded invoke server when enabled by config or env.
func (r *EmbeddedRuntime) Start() error {
	workDir, err := r.deps.resolveRoot()
	if err != nil {
		return err
	}
	cfg, err := r.deps.loadConfig(workDir)
	if err != nil {
		return fmt.Errorf("invoke server: load ftconfig: %w", err)
	}
	if !shouldStartEmbedded(cfg.Server.Embedded) {
		return nil
	}

	serverCfg := Config{
		Host:           embeddedListenHost,
		Port:           effectivePort(cfg.Server),
		CORS:           cfg.Server.CORS,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxRequestSize: cfg.Server.MaxRequestSize,
		Runtime:        "embedded",
		BoundaryRoot:   workDir,
	}
	ApplyListenDefaults(&serverCfg, workDir)
	backend := NewRegistryBackend(r.registryOrNew())
	srv := r.deps.newServer(serverCfg, backend, DefaultEmbeddedVersion(), DefaultLogger())
	if err := srv.StartAsync(); err != nil {
		return fmt.Errorf("invoke server: start: %w", err)
	}
	r.serverMu.Lock()
	r.server = srv
	r.serverMu.Unlock()
	return nil
}

// Stop shuts down the embedded invoke server when running.
func (r *EmbeddedRuntime) Stop() {
	r.serverMu.Lock()
	srv := r.server
	r.server = nil
	r.serverMu.Unlock()
	if srv != nil {
		_ = srv.Stop()
	}
}

func (r *EmbeddedRuntime) startOnce() {
	r.once.Do(func() {
		r.startErr = r.Start()
	})
}

func (r *EmbeddedRuntime) shutdownCh() <-chan struct{} {
	r.shutdownMu.Lock()
	defer r.shutdownMu.Unlock()
	if r.shutdown == nil {
		r.shutdown = make(chan struct{})
	}
	return r.shutdown
}

// WaitForShutdown blocks until a signal or NotifyShutdown.
func (r *EmbeddedRuntime) WaitForShutdown(wait func(<-chan struct{})) {
	wait(r.shutdownCh())
	r.Stop()
}

// NotifyShutdown unblocks WaitForShutdown.
func (r *EmbeddedRuntime) NotifyShutdown() {
	r.shutdownMu.Lock()
	defer r.shutdownMu.Unlock()
	if r.shutdown != nil {
		close(r.shutdown)
		r.shutdown = nil
	}
}

var defaultRuntime = newEmbeddedRuntime(defaultEmbeddedDeps())
