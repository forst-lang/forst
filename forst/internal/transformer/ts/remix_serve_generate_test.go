package transformerts

import (
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/testutil"
)

func TestParseMergedTypecheckProject_remixServeNodeInterop(t *testing.T) {
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", ".."))
	exampleRoot := filepath.Join(projectRoot, "examples", "in", "rfc", "node-interop", "remix-serve")
	mainFT := filepath.Join(exampleRoot, "main.ft")

	log := testutil.TestLogger(t, nil)
	chunks, tc, err := ParseMergedTypecheckProject([]string{mainFT}, log)
	if err != nil {
		t.Fatalf("ParseMergedTypecheckProject: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d", len(chunks))
	}
	if !tc.NeedsNodeRuntime() {
		t.Fatal("expected NeedsNodeRuntime")
	}
}
