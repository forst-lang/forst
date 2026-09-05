package bridgert

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestTerminateHostPID_terminatesSetsidSleepChild(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	if err := TerminateHostPID(pid, 500*time.Millisecond); err != nil {
		t.Fatalf("TerminateHostPID: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected wait error after terminate")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("setsid sleep pid=%d still running after TerminateHostPID", pid)
	}
}
