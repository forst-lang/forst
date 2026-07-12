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
	"strconv"
	"strings"
	"syscall"
	"time"

	"forst/internal/ftconfig"
	"forst/internal/invokeserver"
)

const (
	defaultPortFreeTimeout  = 10 * time.Second
	defaultInvokeReadyWait  = 30 * time.Second
	invokeReadyPollInterval = 100 * time.Millisecond
	maxInvokePortAttempts   = 64
	reloadingMarkerName       = "reloading"
	invokeReadyMarkerName     = "invoke.ready"
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
	return InvokeBaseURLFromHostPort(host, port)
}

// InvokeBaseURLFromHostPort builds http://host:port for invoke health checks.
func InvokeBaseURLFromHostPort(host, port string) string {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = ftconfig.DefaultEmbeddedInvokePort
	}
	return "http://" + net.JoinHostPort(host, port)
}

// PickInvokePort returns the first bindable port >= preferred on host.
func PickInvokePort(cfg *ftconfig.Config, cliPortOverride string) (host, port string, err error) {
	preferred := EffectiveListenPort(cfg, cliPortOverride)
	host = "127.0.0.1"
	if cfg != nil {
		host = cfg.Server.EffectiveInvokeHost()
	}
	port, err = FindNextFreeInvokePort(host, preferred)
	return host, port, err
}

// FindNextFreeInvokePort returns the first bindable TCP port >= preferred.
func FindNextFreeInvokePort(host, preferred string) (string, error) {
	return findNextFreeInvokePortDefault(host, preferred)
}

func findNextFreeInvokePortDefault(host, preferred string) (string, error) {
	if host == "" {
		host = "127.0.0.1"
	}
	start, err := strconv.Atoi(preferred)
	if err != nil || start <= 0 {
		start, _ = strconv.Atoi(ftconfig.DefaultEmbeddedInvokePort)
		if start <= 0 {
			start = 6321
		}
	}
	reserved := reservedInvokePorts()
	for i := 0; i < maxInvokePortAttempts; i++ {
		port := strconv.Itoa(start + i)
		if _, skip := reserved[port]; skip {
			continue
		}
		if portCanListen(host, port) {
			return port, nil
		}
	}
	return "", fmt.Errorf("no free invoke port near %s on %s after %d attempts", preferred, host, maxInvokePortAttempts)
}

func reservedInvokePorts() map[string]struct{} {
	reserved := make(map[string]struct{})
	for _, key := range []string{"PORT", "FORST_DEV_PORT", "VITE_PORT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			reserved[v] = struct{}{}
		}
	}
	return reserved
}

func portCanListen(host, port string) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// ReadInvokeReadyURL returns the invoke base URL from boundaryRoot/.forst/invoke.ready.
func ReadInvokeReadyURL(boundaryRoot string) (string, error) {
	raw, err := os.ReadFile(invokeReadyPath(boundaryRoot))
	if err != nil {
		return "", err
	}
	var payload invokeserver.InvokeReadyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("parse invoke.ready: %w", err)
	}
	if payload.URL == "" {
		return "", fmt.Errorf("invoke.ready missing url")
	}
	return strings.TrimRight(payload.URL, "/"), nil
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

// WaitForInvokeReady polls invoke.ready and /health until the new child is serving.
func WaitForInvokeReady(boundaryRoot, healthURL string, exited <-chan error, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultInvokeReadyWait
	}
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	readyPath := invokeReadyPath(boundaryRoot)
	lastHealthURL := healthURL

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

		if url, err := ReadInvokeReadyURL(boundaryRoot); err == nil && url != "" {
			lastHealthURL = strings.TrimRight(url, "/") + "/health"
		}

		resp, err := client.Get(lastHealthURL)
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
	return fmt.Errorf("invoke server not ready at %s within %v", lastHealthURL, timeout)
}
