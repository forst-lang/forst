package invokeserver

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"forst/internal/ftconfig"
	"forst/internal/invokedispatch"
)

const (
	envInvokeEnabled = "FORST_INVOKE_ENABLED"
	envInvokePort    = "FORST_INVOKE_PORT"
	envBoundaryRoot  = "FORST_BOUNDARY_ROOT"
	invokeReadyFile  = ".forst/invoke.ready"
)

var (
	embeddedOnce   sync.Once
	embeddedErr    error
	globalRegistry *invokedispatch.Registry
	globalServer   *Server
	shutdownCh     chan struct{}
)

// GlobalRegistry returns the registry populated by generated init code.
func GlobalRegistry() *invokedispatch.Registry {
	if globalRegistry == nil {
		globalRegistry = invokedispatch.NewRegistry()
	}
	return globalRegistry
}

// MustStartEmbedded starts the embedded invoke HTTP server once.
// Generated companion files call this from init() when server.embedded is enabled.
func MustStartEmbedded() {
	embeddedOnce.Do(func() {
		embeddedErr = startEmbedded()
	})
	if embeddedErr != nil {
		panic(embeddedErr)
	}
}

func startEmbedded() error {
	workDir, err := resolveBoundaryRoot()
	if err != nil {
		return err
	}
	cfg, err := ftconfig.LoadFromDir(workDir)
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
	}
	backend := NewRegistryBackend(GlobalRegistry())
	globalServer = New(serverCfg, backend, DefaultEmbeddedVersion(), nil)
	if err := globalServer.StartAsync(); err != nil {
		return fmt.Errorf("invoke server: start: %w", err)
	}
	return writeInvokeReady(workDir, serverCfg)
}

func shouldStartEmbedded(ftconfigEnabled bool) bool {
	if ftconfigEnabled {
		return true
	}
	v := os.Getenv(envInvokeEnabled)
	return v == "1" || v == "true"
}

func effectivePort(s ftconfig.ServerConfig) string {
	if p := os.Getenv(envInvokePort); p != "" {
		return p
	}
	return s.EffectiveInvokePort()
}

func resolveBoundaryRoot() (string, error) {
	if root := os.Getenv(envBoundaryRoot); root != "" {
		return filepath.Clean(root), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("invoke server: getwd: %w", err)
	}
	return ftconfig.BoundaryRootFromDir(cwd)
}

type invokeReadyPayload struct {
	URL             string `json:"url"`
	ContractVersion string `json:"contractVersion"`
	Runtime         string `json:"runtime"`
}

func writeInvokeReady(workDir string, cfg Config) error {
	readyPath := filepath.Join(workDir, ".forst", "invoke.ready")
	if strings.Contains(readyPath, "..") {
		return fmt.Errorf("invoke server: invalid ready path")
	}
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		return err
	}
	payload := invokeReadyPayload{
		URL:             cfg.BaseURL(),
		ContractVersion: HTTPContractVersion,
		Runtime:         "embedded",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(readyPath, raw, 0o644)
}

// DefaultEmbeddedVersion returns version metadata for embedded invoke /version.
func DefaultEmbeddedVersion() VersionInfo {
	return VersionInfo{
		Version:         "embedded",
		Commit:          "unknown",
		Date:            "unknown",
		ContractVersion: HTTPContractVersion,
		Runtime:         "embedded",
	}
}

// WaitForShutdown blocks until SIGINT, SIGTERM, or SIGQUIT.
// Call from main in long-lived host-mode binaries to keep the invoke server alive.
func WaitForShutdown() {
	if shutdownCh == nil {
		shutdownCh = make(chan struct{})
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-sigCh:
	case <-shutdownCh:
	}
	if globalServer != nil {
		_ = globalServer.Stop()
	}
}

// NotifyShutdown unblocks WaitForShutdown (for tests).
func NotifyShutdown() {
	if shutdownCh != nil {
		close(shutdownCh)
	}
}
