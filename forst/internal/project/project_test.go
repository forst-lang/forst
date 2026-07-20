package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/modulecheck"
)

func writeMinimalProject(t *testing.T, dir string, extraFiles map[string]string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module projecttest\n\ngo 1.26.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range extraFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProject_Package_returnsMergedNodes(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProject(t, dir, map[string]string{
		"api.ft": "package api\n\nfunc Ping() { return {ok: true} }\n",
	})

	proj, err := Open(nil, OpenOpts{BoundaryRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	unit, err := proj.Package("api")
	if err != nil {
		t.Fatalf("Package: %v", err)
	}
	if unit.Name != "api" {
		t.Fatalf("name = %q", unit.Name)
	}
	if len(unit.Nodes) == 0 {
		t.Fatal("expected merged nodes")
	}
	if len(unit.FilePaths) != 1 {
		t.Fatalf("file paths = %v", unit.FilePaths)
	}
}

func TestProject_Package_unknownName(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProject(t, dir, nil)

	proj, err := Open(nil, OpenOpts{BoundaryRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	_, err = proj.Package("missing")
	if err == nil || !strings.Contains(err.Error(), `package "missing" not found`) {
		t.Fatalf("Package unknown: got %v", err)
	}
}

func TestFilterPackagesUnderBoundary_excludesOutsideFiles(t *testing.T) {
	dir := t.TempDir()
	boundary := filepath.Join(dir, "boundary")
	if err := os.MkdirAll(boundary, 0o755); err != nil {
		t.Fatal(err)
	}
	insideFile := filepath.Join(boundary, "main.ft")
	outsideFile := filepath.Join(dir, "outside", "remote.ft")
	if err := os.MkdirAll(filepath.Dir(outsideFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(insideFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsideFile, []byte("package remote\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proj := &Project{
		BoundaryRoot: boundary,
		Module: &modulecheck.ModuleResult{
			ForstPkgToFiles: map[string][]string{
				"main":   {insideFile},
				"remote": {outsideFile},
			},
		},
	}
	filtered := proj.FilterPackagesUnderBoundary()
	if _, ok := filtered["remote"]; ok {
		t.Fatalf("remote package should be excluded: %v", filtered)
	}
	if paths, ok := filtered["main"]; !ok || len(paths) != 1 || paths[0] != insideFile {
		t.Fatalf("main package should remain: %v", filtered)
	}
}

func TestOpen_resolvesDevProfileFromConfig(t *testing.T) {
	dir := t.TempDir()
	writeMinimalProject(t, dir, nil)
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"dev":{"profile":"runtime"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	proj, err := Open(nil, OpenOpts{BoundaryRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if proj.DevProfile() != DevProfileRuntime {
		t.Fatalf("DevProfile = %q, want runtime", proj.DevProfile())
	}
}
