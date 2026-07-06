package testutil

import (
	"bytes"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// TestLogger returns a logger suitable for tests: ast.SetupTestLogger with output
// discarded unless testing.Verbose().
func TestLogger(tb testing.TB, opts *ast.TestLoggerOptions) *logrus.Logger {
	tb.Helper()
	log := ast.SetupTestLogger(opts)
	if !testing.Verbose() {
		log.SetOutput(bytes.NewBuffer(nil))
	}
	return log
}
