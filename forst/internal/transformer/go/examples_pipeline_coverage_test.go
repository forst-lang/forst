package transformergo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPipeline_examplesInKnownGood runs the full compile pipeline on example programs that
// compile standalone (same set as internal/compiler TestProgramCompilation plus a few guards).
func TestPipeline_examplesInKnownGood(t *testing.T) {
	t.Parallel()
	examplesRoot := filepath.Join("..", "..", "..", "..", "examples", "in")
	relPaths := []string{
		"basic.ft",
		"loop.ft",
		"union_error_types.ft",
		"union_error_narrowing.ft",
		"result_ensure.ft",
		"generics.ft",
		"ensure.ft",
		"pointers.ft",
		"go_builtins.ft",
		"basic_function.ft",
		"result_if.ft",
		"nominal_error.ft",
		"echo.ft",
		"rfc/guard/shape_guard.ft",
		"rfc/guard/basic_guard.ft",
	}
	for _, rel := range relPaths {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(examplesRoot, rel)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read %s: %v", p, err)
			}
			out := compileForstPipeline(t, string(src))
			if !strings.Contains(out, "package ") {
				t.Fatalf("unexpected output for %s", rel)
			}
		})
	}
}
