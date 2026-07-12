package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestStopOpts_narrowKillPreservesSetsidPeer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix process groups")
	}

	peerPIDPath := t.TempDir() + "/peer.pid"
	peerDone := make(chan struct{})
	go func() {
		cmd := exec.Command("sleep", "30")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if err := cmd.Start(); err != nil {
			close(peerDone)
			return
		}
		_ = os.WriteFile(peerPIDPath, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0o644)
		_ = cmd.Wait()
		close(peerDone)
	}()

	deadline := time.Now().Add(2 * time.Second)
	var peerPID int
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(peerPIDPath)
		if err == nil && len(raw) > 0 {
			_, _ = fmt.Sscanf(string(raw), "%d", &peerPID)
			if peerPID > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if peerPID <= 0 {
		t.Fatal("setsid peer did not start")
	}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(outPath, []byte("package main\nfunc main() { select {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proc, err := StartGoProgram(outPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Stop(DefaultGoProgramStopGrace(), StopOpts{NarrowKill: true}); err != nil {
		t.Fatal(err)
	}

	peerProc, err := os.FindProcess(peerPID)
	if err != nil {
		t.Fatal(err)
	}
	if err := peerProc.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("setsid peer pid=%d should survive narrow parent stop: %v", peerPID, err)
	}
	_ = peerProc.Kill()
	<-peerDone
}

func TestStopOpts_groupKillTerminatesChild(t *testing.T) {
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
	case <-time.After(15 * time.Second):
		t.Fatal("timed out stopping go program")
	}
}

func TestReloadStopGrace_attachOnlyUsesShorterGrace(t *testing.T) {
	t.Setenv("FORST_NODE_ATTACH_ONLY", "1")
	if ReloadStopGrace() != reloadStopGrace {
		t.Fatalf("expected %v, got %v", reloadStopGrace, ReloadStopGrace())
	}
	t.Setenv("FORST_NODE_ATTACH_ONLY", "")
	if ReloadStopGrace() != defaultGoProgramStopGrace {
		t.Fatalf("expected default grace %v, got %v", defaultGoProgramStopGrace, ReloadStopGrace())
	}
}
