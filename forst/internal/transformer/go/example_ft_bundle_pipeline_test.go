package transformergo

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPipeline_examplesBundle runs compileForstPipeline on the same large example set as
// internal/typechecker TestCheckTypes_examplesBundle exercises the same example bundle for the transformer.
func TestPipeline_examplesBundle(t *testing.T) {
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
	}
	root := filepath.Join("..", "..", "..", "..", "examples", "in")
	for _, name := range rel {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(root, name)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			_ = compileForstPipeline(t, string(src))
		})
	}
}
