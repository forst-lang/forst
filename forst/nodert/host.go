package nodert

// Host mode invariants (local trust zone only):
//   - One Go supervisor client per process; the host rejects a second socket connection.
//   - RPC uses a single bidirectional net.Conn (Unix domain socket or loopback TCP on Windows).
//   - Host child stdout/stderr are piped and forwarded to the parent process (bootstrap keeps stdout for RPC).
//   - Manifest allowlist and path rules are unchanged; initialize is required before calls.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	logrus "github.com/sirupsen/logrus"
)

type hostReadyMarker struct {
	PID     int    `json:"pid"`
	Socket  string `json:"socket"`
	TCPPort int    `json:"tcpPort"`
	Phase   string `json:"phase"`
}

func hostMarkerReadyForRPC(marker hostReadyMarker) bool {
	phase := strings.TrimSpace(marker.Phase)
	if phase == "" {
		return true
	}
	return phase == "app"
}

func waitForHostReady(ctx context.Context, socketPath, readyPath string) (net.Conn, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		select {
		case <-ctx.Done():
			msg := fmt.Sprintf("node runtime: host ready timeout (socket=%s)", socketPath)
			if readyPath != "" {
				if data, err := os.ReadFile(readyPath); err == nil {
					msg += fmt.Sprintf("; ready=%s", string(data))
				}
			}
			if lastErr != nil {
				msg += fmt.Sprintf("; last_error=%v", lastErr)
			}
			return nil, fmt.Errorf("%s", msg)
		case <-ticker.C:
			dialPath := socketPath
			if readyPath != "" {
				marker, ok := readHostReadyMarker(readyPath)
				if !ok || !hostMarkerReadyForRPC(marker) {
					continue
				}
				if !processAlive(marker.PID) {
					lastErr = fmt.Errorf("ready pid %d not alive", marker.PID)
					continue
				}
				if runtime.GOOS == "windows" && marker.TCPPort > 0 {
					dialPath = fmt.Sprintf("127.0.0.1:%d", marker.TCPPort)
				} else if marker.Socket != "" {
					dialPath = marker.Socket
				}
			}
			conn, err := dialHostSocket(dialPath)
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
	}
}

func readHostReadyMarker(readyPath string) (hostReadyMarker, bool) {
	data, err := os.ReadFile(readyPath)
	if err != nil {
		return hostReadyMarker{}, false
	}
	var marker hostReadyMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return hostReadyMarker{}, false
	}
	return marker, true
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func dialHostSocket(socketPath string) (net.Conn, error) {
	if socketPath == "" {
		return nil, fmt.Errorf("host socket path is empty")
	}
	if runtime.GOOS == "windows" || looksLikeTCPAddr(socketPath) {
		addr := socketPath
		if len(addr) > 6 && addr[:6] == "tcp://" {
			addr = addr[6:]
		}
		return net.DialTimeout("tcp", addr, 500*time.Millisecond)
	}
	return net.DialTimeout("unix", socketPath, 500*time.Millisecond)
}

func looksLikeTCPAddr(path string) bool {
	if len(path) > 6 && path[:6] == "tcp://" {
		return true
	}
	host, port, err := net.SplitHostPort(path)
	return err == nil && host != "" && port != ""
}

func spawnHostProcess(cmd HostSpawnCommand, workDir string, log *logrus.Logger) (*managedProcess, error) {
	execCmd := exec.Command(cmd.Executable, cmd.Args...)
	if workDir != "" {
		execCmd.Dir = workDir
	}
	execCmd.Env = cmd.Env

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr, err := execCmd.StderrPipe()
	if err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := execCmd.Start(); err != nil {
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("start host process (executable=%s): %w", cmd.Executable, err)
	}

	if log == nil {
		log = logrus.New()
		log.SetLevel(logrus.ErrorLevel)
	}
	log.WithFields(logrus.Fields{
		"component":       "nodert",
		"event":           "spawn_host",
		"node_pid":        execCmd.Process.Pid,
		"node_executable": cmd.Executable,
		"socket_path":     cmd.SocketPath,
	}).Debug("spawned host app process")

	go forwardChildOutput(stdout, os.Stdout, log, "stdout")
	go forwardChildOutput(stderr, os.Stderr, log, "stderr")

	return &managedProcess{
		cmd:    execCmd,
		stdout: stdout,
		stderr: stderr,
		log:    log,
	}, nil
}

func cleanupHostSocketFiles(socketPath, readyPath string) {
	if socketPath != "" {
		_ = os.Remove(socketPath)
	}
	if readyPath != "" {
		_ = os.Remove(readyPath)
	}
}
