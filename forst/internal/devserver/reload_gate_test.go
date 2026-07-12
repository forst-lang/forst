package devserver

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMarkReloading_writesAndClearsMarker(t *testing.T) {
	dir := t.TempDir()
	if err := MarkReloading(dir, true, 7); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(reloadingMarkerPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	var marker ReloadMarker
	if err := json.Unmarshal(raw, &marker); err != nil {
		t.Fatal(err)
	}
	if !marker.Reloading || marker.Generation != 7 {
		t.Fatalf("marker=%+v", marker)
	}
	if err := MarkReloading(dir, false, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(reloadingMarkerPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("expected marker removed, err=%v", err)
	}
}

func TestWaitPortFree_detectsReleasedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := waitPortFree(addr, 50*time.Millisecond); err == nil {
		t.Fatal("expected timeout while port is bound")
	}
	ln.Close()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if canListen(addr) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("port %s not reusable after close", addr)
}

func canListen(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func TestFindNextFreeInvokePort_skipsBoundPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, boundPort, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	next, err := FindNextFreeInvokePort("127.0.0.1", boundPort)
	if err != nil {
		t.Fatal(err)
	}
	if next == boundPort {
		t.Fatalf("expected port after %s, got same port", boundPort)
	}
	if !portCanListen("127.0.0.1", next) {
		t.Fatalf("picked port %s is not bindable", next)
	}
}

func TestReadInvokeReadyURL_parsesPayload(t *testing.T) {
	dir := t.TempDir()
	path := invokeReadyPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"url":"http://127.0.0.1:6323","contractVersion":"1","runtime":"embedded"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadInvokeReadyURL(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://127.0.0.1:6323" {
		t.Fatalf("url=%q", got)
	}
}

func TestRemoveInvokeReady(t *testing.T) {
	dir := t.TempDir()
	path := invokeReadyPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveInvokeReady(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected invoke.ready removed, err=%v", err)
	}
}

func TestWaitForInvokeReady_usesInvokeReadyURL(t *testing.T) {
	dir := t.TempDir()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	baseURL := "http://" + ln.Addr().String()
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"success\":true}"))
			_ = conn.Close()
		}
	}()

	readyPath := invokeReadyPath(dir)
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"url": baseURL})
	if err := os.WriteFile(readyPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	var exited <-chan error
	if err := WaitForInvokeReady(dir, "http://127.0.0.1:1/health", exited, time.Second); err != nil {
		t.Fatalf("WaitForInvokeReady: %v", err)
	}
}
