package compiler

import (
	"os"
	"path/filepath"
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
		done <- proc.Stop(DefaultGoProgramStopGrace())
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
