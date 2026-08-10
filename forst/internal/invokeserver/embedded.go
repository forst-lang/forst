package invokeserver

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"forst/internal/ftconfig"
	"forst/internal/invokedispatch"
)

const (
	envInvokeEnabled = "FORST_INVOKE_ENABLED"
	// EnvInvokePort overrides the embedded invoke listen port (dev reload port pick).
	EnvInvokePort   = "FORST_INVOKE_PORT"
	envInvokePort   = EnvInvokePort
	envBoundaryRoot = "FORST_BOUNDARY_ROOT"
)

// GlobalRegistry returns the registry populated by generated init code.
func GlobalRegistry() *invokedispatch.Registry {
	return defaultRuntime.registryOrNew()
}

// MustStartEmbedded starts the embedded invoke HTTP server once.
// Generated companion files call this from init() when server.embedded is enabled.
func MustStartEmbedded() {
	defaultRuntime.startOnce()
	if defaultRuntime.startErr != nil {
		panic(defaultRuntime.startErr)
	}
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

// InvokeReadyPayload is written to boundaryRoot/.forst/invoke.ready when embedded invoke starts.
type InvokeReadyPayload struct {
	URL             string `json:"url"`
	SocketPath      string `json:"socketPath,omitempty"`
	Generation      uint64 `json:"generation,omitempty"`
	PID             int    `json:"pid,omitempty"`
	ContractVersion string `json:"contractVersion"`
	Runtime         string `json:"runtime"`
}

func writeInvokeReady(workDir string, cfg Config, generation uint64) error {
	readyPath := filepath.Join(workDir, ".forst", "invoke.ready")
	if strings.Contains(readyPath, "..") {
		return fmt.Errorf("invoke server: invalid ready path")
	}
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o750); err != nil {
		return err
	}
	payload := InvokeReadyPayload{
		URL:             cfg.BaseURL(),
		SocketPath:      cfg.SocketPath,
		Generation:      generation,
		PID:             os.Getpid(),
		ContractVersion: HTTPContractVersion,
		Runtime:         cfg.Runtime,
	}
	if cfg.network() == transportUnix {
		payload.URL = ""
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tmp := readyPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, readyPath)
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
	defaultRuntime.WaitForShutdown(func(shutdown <-chan struct{}) {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		select {
		case <-sigCh:
		case <-shutdown:
		}
	})
}

// NotifyShutdown unblocks WaitForShutdown (for tests).
func NotifyShutdown() {
	defaultRuntime.NotifyShutdown()
}
