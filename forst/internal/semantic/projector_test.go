package semantic_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/semantic"

	"github.com/sirupsen/logrus"
)

func TestBuildSnapshot_constraintsGolden(t *testing.T) {
	runSnapshotGolden(t, "constraints")
}

func TestBuildSnapshot_routerGolden(t *testing.T) {
	runSnapshotGolden(t, "router")
}

func TestBuildSnapshot_layoutGolden(t *testing.T) {
	runSnapshotGolden(t, "layout")
}

func runSnapshotGolden(t *testing.T, name string) {
	t.Helper()
	root := filepath.Join("testdata", name)
	boundaryRoot := mustAbs(t, root)
	files, err := collectFtFiles(boundaryRoot)
	if err != nil {
		t.Fatalf("collect files: %v", err)
	}
	log := logrus.New()
	log.SetOutput(os.Stderr)
	snap, err := semantic.BuildSnapshot(files, boundaryRoot, log)
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	normalizeSnapshotForGolden(snap, name)
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	golden := filepath.Join(root, "snapshot.golden.json")
	if os.Getenv("UPDATE_SEMANTIC_GOLDENS") == "1" {
		if err := os.WriteFile(golden, append(data, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with UPDATE_SEMANTIC_GOLDENS=1)", golden, err)
	}
	if string(want) != string(append(data, '\n')) {
		t.Fatalf("snapshot mismatch for %s (run UPDATE_SEMANTIC_GOLDENS=1)", name)
	}
}

func collectFtFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ft") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func normalizeSnapshotForGolden(snap *semantic.GenerateRequest, fixtureName string) {
	snap.CompilerVersion = "test"
	snap.Module.Root = "testdata/" + fixtureName
	snap.Module.GoModule = "forst"
	for i := range snap.Packages {
		if snap.Packages[i].FunctionIDs == nil {
			snap.Packages[i].FunctionIDs = []string{}
		}
		if snap.Packages[i].TypeIDs == nil {
			snap.Packages[i].TypeIDs = []string{}
		}
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	if !filepath.IsAbs(path) {
		_, file, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("runtime.Caller failed")
		}
		path = filepath.Join(filepath.Dir(file), path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	return abs
}
