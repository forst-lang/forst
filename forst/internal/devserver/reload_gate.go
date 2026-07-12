package devserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"forst/internal/ftconfig"
)

const (
	defaultPortFreeTimeout   = 10 * time.Second
	defaultInvokeReadyWait   = 30 * time.Second
	invokeReadyPollInterval  = 100 * time.Millisecond
	reloadingMarkerName      = "reloading"
	invokeReadyMarkerName    = "invoke.ready"
)

// ReloadMarker is written to boundaryRoot/.forst/reloading during hot reload.
type ReloadMarker struct {
	Reloading  bool   `json:"reloading"`
	Generation uint64 `json:"generation"`
}

func reloadingMarkerPath(boundaryRoot string) string {
	return filepath.Join(filepath.Clean(boundaryRoot), ".forst", reloadingMarkerName)
}

func invokeReadyPath(boundaryRoot string) string {
	return filepath.Join(filepath.Clean(boundaryRoot), ".forst", invokeReadyMarkerName)
}

// MarkReloading writes or clears the parent-side reload marker.
func MarkReloading(boundaryRoot string, reloading bool, generation uint64) error {
	path := reloadingMarkerPath(boundaryRoot)
	if !reloading {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(ReloadMarker{Reloading: true, Generation: generation})
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// RemoveInvokeReady deletes invoke.ready so clients do not treat a stale server as ready.
func RemoveInvokeReady(boundaryRoot string) error {
	err := os.Remove(invokeReadyPath(boundaryRoot))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// InvokeListenAddr returns host:port for the embedded invoke server.
func InvokeListenAddr(cfg *ftconfig.Config) string {
	if cfg == nil {
		return "127.0.0.1:" + ftconfig.DefaultEmbeddedInvokePort
	}
	return cfg.Server.EffectiveInvokeHost() + ":" + cfg.Server.EffectiveInvokePort()
}

// InvokeBaseURL returns http://host:port for health polling.
func InvokeBaseURL(cfg *ftconfig.Config) string {
	host, port, _ := net.SplitHostPort(InvokeListenAddr(cfg))
	if host == "" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// WaitPortFree blocks until nothing accepts TCP connections on addr.
func WaitPortFree(addr string, timeout time.Duration) error {
	return portFreeWaiter(addr, timeout)
}

// portFreeWaiter may be replaced in tests when the default invoke port is occupied.
var portFreeWaiter = waitPortFree

func waitPortFree(addr string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultPortFreeTimeout
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if portAppearsFree(addr) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("port %s still in use after %v", addr, timeout)
}

func portAppearsFree(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 150*time.Millisecond)
	if err != nil {
		return isConnRefused(err)
	}
	_ = conn.Close()
	return false
}

func isConnRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) ||
		strings.Contains(strings.ToLower(err.Error()), "connection refused")
}

// invokeReadyWaiter may be replaced in tests to avoid polling a real HTTP server.
var invokeReadyWaiter = WaitForInvokeReady

// WaitForInvokeReady polls /health and invoke.ready until the new child is serving.
func WaitForInvokeReady(boundaryRoot, healthURL string, exited <-chan error, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultInvokeReadyWait
	}
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	readyPath := invokeReadyPath(boundaryRoot)

	for time.Now().Before(deadline) {
		select {
		case err := <-exited:
			if err != nil {
				return fmt.Errorf("child exited before invoke ready: %w", err)
			}
			return fmt.Errorf("child exited before invoke ready")
		default:
		}

		if _, err := os.Stat(readyPath); err != nil {
			time.Sleep(invokeReadyPollInterval)
			continue
		}

		resp, err := client.Get(healthURL)
		if err != nil {
			time.Sleep(invokeReadyPollInterval)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(invokeReadyPollInterval)
			continue
		}

		var health struct {
			Success bool `json:"success"`
		}
		if json.Unmarshal(body, &health) != nil || !health.Success {
			time.Sleep(invokeReadyPollInterval)
			continue
		}
		return nil
	}
	return fmt.Errorf("invoke server not ready at %s within %v", healthURL, timeout)
}
