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
