package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestParseFile_examplesBundle parses large example files to cover parser edge paths.
func TestParseFile_examplesBundle(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	rel := []string{
		"basic.ft",
		"basic_function.ft",
		"loop.ft",
		"generics.ft",
		"go_builtins.ft",
		"slices.ft",
		"go_interop/cli.ft",
		"ensure.ft",
		"result_ensure.ft",
		"result_if.ft",
		"pointers.ft",
		"map_catalog.ft",
		"echo.ft",
		"union_error_types.ft",
		"union_error_narrowing.ft",
		"nominal_error.ft",
		"imports/main.ft",
		"imports/cli.ft",
		"imports/greeting.ft",
		"imports/fmt_only.ft",
		"rfc/sidecar/tests/echo.ft",
		"rfc/sidecar/tests/type_safety.ft",
		"rfc/sidecar/tests/user.ft",
		"rfc/guard/shape_guard.ft",
		"rfc/guard/basic_guard.ft",
		"tictactoe/main/server.ft",
		"tictactoe/main/engine.ft",
		"rfc/providers/providers.ft",
		"rfc/providers/main_wiring.ft",
		"rfc/providers/cross_pkg/auth/log.ft",
		"rfc/providers/cross_pkg/api/handle.ft",
	}
	root := filepath.Join("..", "..", "..", "examples", "in")
	for _, name := range rel {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(root, name)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			pr := NewTestParser(string(src), log)
			if _, err := pr.ParseFile(); err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
		})
	}
}
