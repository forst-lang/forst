package transformergo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// examplesPipelineSkip lists example paths excluded from the known-good pipeline until fixed.
var examplesPipelineSkip = map[string]string{}

// TestWriteUnionErrorNarrowingGolden regenerates examples/out/union_error_narrowing.go.
// Run: UPDATE_UNION_ERROR_NARROWING_GOLDEN=1 go test ./internal/transformer/go -run TestWriteUnionErrorNarrowingGolden -count=1
func TestWriteUnionErrorNarrowingGolden(t *testing.T) {
	if os.Getenv("UPDATE_UNION_ERROR_NARROWING_GOLDEN") != "1" && os.Getenv("UPDATE_EXAMPLES_GOLDENS") != "1" {
		t.Skip("set UPDATE_UNION_ERROR_NARROWING_GOLDEN=1 or UPDATE_EXAMPLES_GOLDENS=1 to regenerate golden")
	}
	examplesRoot := filepath.Join("..", "..", "..", "..", "examples", "in")
	src, err := os.ReadFile(filepath.Join(examplesRoot, "union_error_narrowing.ft"))
	if err != nil {
		t.Fatal(err)
	}
	out := compileForstPipeline(t, string(src))
	goldenPath := filepath.Join("..", "..", "..", "..", "examples", "out", "union_error_narrowing.go")
	if err := os.WriteFile(goldenPath, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d bytes)", goldenPath, len(out))
}

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
		"map_catalog.ft",
		"rfc/guard/anonymous_objects.ft",
		"tictactoe/engine.ft",
		"rfc/guard/shape_guard.ft",
		"rfc/guard/basic_guard.ft",
	}
	for _, rel := range relPaths {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			if reason, skip := examplesPipelineSkip[rel]; skip {
				t.Skip(reason)
			}
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
