// unix_listener binds invoke RPC on a mode-0600 Unix domain socket with stale-socket cleanup.
package invokeserver

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
)

// listenUnixSocket creates path's parent dir, removes stale sockets, listens, and chmods 0600.
func listenUnixSocket(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("invoke unix socket: mkdir: %w", err)
	}
	if err := removeStaleSocket(path, processAlive, readReadyMarkerPID); err != nil {
		return nil, err
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("invoke unix socket: listen: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("invoke unix socket: chmod: %w", err)
	}
	return ln, nil
}

// removeStaleSocket unlinks path when no live owner is reported by readMarkerPID.
func removeStaleSocket(path string, isOwnerAlive func(pid int) bool, readMarkerPID func() (int, bool)) error {
	if readMarkerPID != nil {
		if pid, ok := readMarkerPID(); ok && isOwnerAlive(pid) {
			return fmt.Errorf("invoke unix socket: owner pid %d still alive", pid)
		}
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("invoke unix socket: remove stale: %w", err)
	}
	return nil
}

// processAlive reports whether pid is running (signal 0 probe).
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// readReadyMarkerPID is a hook for tests; production returns no owner pid.
func readReadyMarkerPID() (int, bool) {
	return 0, false
}
