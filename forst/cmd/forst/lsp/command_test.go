package lsp

import (
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestStartLSPServer_helperProcess(_ *testing.T) {
	if os.Getenv("FORST_LSP_HELPER_PROCESS") != "1" {
		return
	}
	log := logrus.New()
	log.SetOutput(io.Discard)
	// Invalid port forces server.Start() to fail immediately and StartLSPServer to exit(1).
	StartLSPServer("invalid-port", log)
}

func TestStartLSPServer_exitOnStartError(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestStartLSPServer_helperProcess")
	cmd.Env = append(os.Environ(), "FORST_LSP_HELPER_PROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected helper process to exit with non-zero status")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
}
