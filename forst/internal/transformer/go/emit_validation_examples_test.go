package transformergo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEmitValidation_examplesBundle compiles representative examples/in/*.ft through the full pipeline.
func TestEmitValidation_examplesBundle(t *testing.T) {
	t.Parallel()
	rel := []string{
		"basic.ft",
		"loop.ft",
		"echo.ft",
		"basic_function.ft",
		"result_if.ft",
		"result_ensure.ft",
		"generics.ft",
		"go_builtins.ft",
		"ensure.ft",
		"pointers.ft",
		"union_error_types.ft",
		"union_error_narrowing.ft",
		"nominal_error.ft",
		"map_catalog.ft",
		"rfc/guard/basic_guard.ft",
		"rfc/guard/shape_guard.ft",
	}
	root := examplesInRoot(t)
	for _, name := range rel {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(root, name)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			out := compileForstPipeline(t, string(src))
			assertGoParses(t, out)
		})
	}
}
