package devserver

import (
	"io"
	"testing"

	"forst/internal/compiler"

	"github.com/sirupsen/logrus"
)

type mockConfig struct{}

func (m *mockConfig) FindForstFiles(root string) ([]string, error) {
	return nil, nil
}

func testDevServer(t *testing.T) *DevServer {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := compiler.Args{
		LogLevel:           "info",
		ReportPhases:       false,
		ReportMemoryUsage:  false,
		ExportStructFields: false,
	}
	comp := compiler.New(args, log)
	cfg := &mockConfig{}
	return NewHTTPServer(
		"0",
		comp,
		log,
		cfg,
		BuildInfo{Version: "dev", Commit: "unknown", Date: "unknown"},
		HTTPOpts{ReadTimeoutSec: 1, WriteTimeoutSec: 1, CORS: true},
		t.TempDir(),
	)
}

func TestNewHTTPServer_initializesTypesCache(t *testing.T) {
	s := testDevServer(t)
	if s.typesCache == nil {
		t.Fatal("typesCache must be non-nil for /types caching")
	}
}
