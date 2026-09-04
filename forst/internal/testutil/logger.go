package testutil

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// TestLogger returns a quiet compiler logger for tests (see ast.SetupTestLoggerFor).
//
//	FORST_TEST_LOG=1     → Debug on stderr
//	FORST_TEST_LOG=fail  → buffer Debug; print via t.Log only if the test fails
func TestLogger(tb testing.TB, opts *ast.TestLoggerOptions) *logrus.Logger {
	tb.Helper()
	return ast.SetupTestLoggerFor(tb, opts)
}
