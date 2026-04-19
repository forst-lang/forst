package lsp

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestStartLSPServer_returnsErrorOnStartFailure(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	err := StartLSPServer("invalid-port", log)
	if err == nil {
		t.Fatal("expected error when listen address is invalid")
	}
}
