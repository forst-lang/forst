package compiler

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStartGoProgram_startsAndStops(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(outPath, []byte("package main\nfunc main() { select {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proc, err := StartGoProgram(outPath, "")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- proc.Stop(DefaultGoProgramStopGrace(), StopOpts{})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Stop: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out stopping go program")
	}
}

func TestStartGoProgram_stopReleasesListeningPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "main.go")
	src := `package main
import (
	"net/http"
	"os"
)
func main() {
	port := os.Getenv("TEST_LISTEN_PORT")
	http.ListenAndServe("127.0.0.1:" + port, nil)
}
`
	if err := os.WriteFile(outPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := newGoRunCommand(outPath, "")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Env = append(os.Environ(), fmt.Sprintf("TEST_LISTEN_PORT=%d", port))

	proc := &GoProgramProcess{cmd: cmd, done: make(chan error, 1)}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		proc.done <- cmd.Wait()
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond); dialErr != nil {
		t.Fatal("expected child to bind port before stop")
	}

	if err := proc.Stop(DefaultGoProgramStopGrace(), StopOpts{}); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	deadline = time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if dialErr != nil {
			if isConnRefused(dialErr) {
				return
			}
			t.Fatalf("unexpected dial error: %v", dialErr)
		}
		_ = conn.Close()
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("port %s still in use after Stop", addr)
}

func isConnRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) ||
		strings.Contains(strings.ToLower(err.Error()), "connection refused")
}
