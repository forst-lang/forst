package nodert

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReadHostMarkerPID(t *testing.T) {
	dir := t.TempDir()
	if pid := ReadHostMarkerPID(dir); pid != 0 {
		t.Fatalf("expected 0 without marker, got %d", pid)
	}

	readyPath := filepath.Join(dir, ".forst", "node.sock.ready")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, []byte(fmt.Sprintf(`{"pid":%d,"phase":"app"}`, os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if pid := ReadHostMarkerPID(dir); pid != os.Getpid() {
		t.Fatalf("ReadHostMarkerPID = %d want %d", pid, os.Getpid())
	}
}

func TestReattachSkipReason(t *testing.T) {
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "node.sock.ready")

	if got := ReattachSkipReason(readyPath); got != "marker_missing" {
		t.Fatalf("got %q want marker_missing", got)
	}

	if err := os.WriteFile(readyPath, []byte(`{"pid":999999999,"phase":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReattachSkipReason(readyPath); got == "" || got == "marker_missing" {
		t.Fatalf("got %q want pid_dead", got)
	}

	if err := os.WriteFile(readyPath, []byte(fmt.Sprintf(`{"pid":%d,"phase":"listening"}`, os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReattachSkipReason(readyPath); got == "" || got == "marker_missing" {
		t.Fatalf("got %q want phase_not_ready", got)
	}

	if err := os.WriteFile(readyPath, []byte(fmt.Sprintf(`{"pid":%d,"phase":"app"}`, os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ReattachSkipReason(readyPath); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
