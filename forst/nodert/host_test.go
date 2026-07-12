package nodert

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWaitForHostReady_success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket test")
	}

	dir := t.TempDir()
	socketPath := filepath.Join(dir, "node.sock")
	readyPath := socketPath + ".ready"

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		_ = conn.Close()
	}()

	if err := os.WriteFile(readyPath, []byte(`{"pid":`+strconv.Itoa(os.Getpid())+`,"socket":"`+socketPath+`","phase":"app"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := waitForHostReady(ctx, socketPath, readyPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()
}

func TestConnectExistingHost_success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket test")
	}

	dir := t.TempDir()
	socketPath := filepath.Join(dir, "node.sock")
	readyPath := socketPath + ".ready"

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		_ = conn.Close()
	}()

	if err := os.WriteFile(readyPath, []byte(`{"pid":`+strconv.Itoa(os.Getpid())+`,"socket":"`+socketPath+`","phase":"app"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, ok, err := connectExistingHost(ctx, socketPath, readyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || conn == nil {
		t.Fatal("expected existing host connection")
	}
	_ = conn.Close()
}

func TestConnectExistingHost_absent(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "node.sock")
	readyPath := socketPath + ".ready"

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	conn, ok, err := connectExistingHost(ctx, socketPath, readyPath)
	if err != nil {
		t.Fatal(err)
	}
	if ok || conn != nil {
		t.Fatalf("expected no attach, ok=%v conn=%v", ok, conn)
	}
}

func TestWaitForHostReady_ignoresListeningPhase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket test")
	}

	dir := t.TempDir()
	socketPath := filepath.Join(dir, "node.sock")
	readyPath := socketPath + ".ready"

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()

	if err := os.WriteFile(readyPath, []byte(`{"pid":`+strconv.Itoa(os.Getpid())+`,"socket":"`+socketPath+`","phase":"listening"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	_, err = waitForHostReady(ctx, socketPath, readyPath)
	if err == nil {
		t.Fatal("expected timeout while phase is listening")
	}
	if !strings.Contains(err.Error(), "host ready timeout") {
		t.Fatalf("err = %v", err)
	}
}

func TestHostMarkerReadyForRPC(t *testing.T) {
	if !hostMarkerReadyForRPC(hostReadyMarker{Phase: ""}) {
		t.Fatal("empty phase should be ready")
	}
	if !hostMarkerReadyForRPC(hostReadyMarker{Phase: "app"}) {
		t.Fatal("app phase should be ready")
	}
	if hostMarkerReadyForRPC(hostReadyMarker{Phase: "listening"}) {
		t.Fatal("listening phase should not be ready")
	}
}

func TestWaitForHostReady_timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket test")
	}

	dir := t.TempDir()
	socketPath := filepath.Join(dir, "missing.sock")

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	_, err := waitForHostReady(ctx, socketPath, "")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "host ready timeout") {
		t.Fatalf("err = %v", err)
	}
}
