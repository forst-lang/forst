package nodert

import (
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestManagedProcess_terminateSigkillReturnsNil(t *testing.T) {
	cmd := exec.Command("sleep", strconv.Itoa(120))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	proc := &managedProcess{cmd: cmd}
	t.Cleanup(func() { _ = proc.terminate() })

	if err := proc.terminate(); err != nil {
		t.Fatalf("terminate after SIGKILL: %v", err)
	}
}

func TestManagedProcess_terminateExitsQuicklyWhenChildIgnoresStdinClose(t *testing.T) {
	cmd := exec.Command("sleep", strconv.Itoa(120))
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	proc := &managedProcess{cmd: cmd}

	done := make(chan error, 1)
	go func() { done <- proc.terminate() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("terminate: %v", err)
		}
	case <-time.After(shutdownGracePeriod + 3*time.Second):
		t.Fatal("terminate did not finish within grace + kill window")
	}
}
