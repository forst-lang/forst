package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestGenerateLink_createsSymlinkForScopedName(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	log := discardLinkLog()

	if err := linkGeneratedClient(boundary, outDir, "@forst/gen", log); err != nil {
		t.Fatalf("linkGeneratedClient: %v", err)
	}

	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("eval symlink: %v", err)
	}
	want, err := filepath.EvalSymlinks(outDir)
	if err != nil {
		want = outDir
	}
	if resolved != want {
		t.Fatalf("link target = %q, want %q", resolved, want)
	}
}

func TestGenerateLink_leavesCorrectLinkUntouched(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	log := discardLinkLog()
	packageName := "@forst/gen"
	linkPath := filepath.Join(nodeModules, "@forst", "gen")

	if err := linkGeneratedClient(boundary, outDir, packageName, log); err != nil {
		t.Fatalf("initial link: %v", err)
	}
	if err := writeForstGeneratedMarker(outDir, boundary, packageName, nil); err != nil {
		t.Fatal(err)
	}

	var removeCalls, symlinkCalls, writeCalls int
	origRemove := generateLinkIO.RemoveAll
	origSymlink := generateLinkIO.Symlink
	origWrite := generateLinkIO.WriteFile
	origJunction := generateLinkIO.Junction
	origCopy := generateLinkIO.CopyDir
	t.Cleanup(func() {
		generateLinkIO.RemoveAll = origRemove
		generateLinkIO.Symlink = origSymlink
		generateLinkIO.WriteFile = origWrite
		generateLinkIO.Junction = origJunction
		generateLinkIO.CopyDir = origCopy
	})
	generateLinkIO.RemoveAll = func(path string) error {
		removeCalls++
		return origRemove(path)
	}
	generateLinkIO.Symlink = func(target, link string) error {
		symlinkCalls++
		return origSymlink(target, link)
	}
	generateLinkIO.WriteFile = func(name string, data []byte, perm os.FileMode) error {
		writeCalls++
		return origWrite(name, data, perm)
	}
	generateLinkIO.Junction = func(_, _ string) error {
		symlinkCalls++
		return errors.New("junction unused on this test host")
	}
	generateLinkIO.CopyDir = func(_, _ string) error {
		t.Fatal("CopyDir must not run when link is already correct")
		return nil
	}

	var buf bytes.Buffer
	log = logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	if err := linkGeneratedClient(boundary, outDir, packageName, log); err != nil {
		t.Fatalf("second link: %v", err)
	}
	if removeCalls != 0 || symlinkCalls != 0 || writeCalls != 0 {
		t.Fatalf("expected no filesystem writes, got remove=%d symlink=%d write=%d", removeCalls, symlinkCalls, writeCalls)
	}
	if !strings.Contains(buf.String(), "unchanged") {
		t.Fatalf("expected action unchanged in log, got %q", buf.String())
	}
	if _, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("link missing after untouched pass: %v", err)
	}
}

func TestGenerateLink_replacesLinkPointingElsewhere(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	packageName := "@forst/gen"

	otherOut := filepath.Join(boundary, ".forst", "other-client")
	if err := os.MkdirAll(otherOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeForstGeneratedMarker(otherOut, boundary, packageName, nil); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	linkFixtureTarget(t, otherOut, linkPath)

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	if err := linkGeneratedClient(boundary, outDir, packageName, log); err != nil {
		t.Fatalf("linkGeneratedClient: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	want, err := filepath.EvalSymlinks(outDir)
	if err != nil {
		want = outDir
	}
	if resolved != want {
		t.Fatalf("after replace link=%q want=%q", resolved, want)
	}
	if !strings.Contains(buf.String(), "replaced") {
		t.Fatalf("expected action replaced, got %q", buf.String())
	}
}

func TestGenerateLink_refusesToReplaceRegularFile(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	log := discardLinkLog()
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(linkPath, []byte("not a package"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := linkGeneratedClient(boundary, outDir, "@forst/gen", log)
	if err == nil {
		t.Fatal("expected error for regular file at link path")
	}
	if !strings.Contains(err.Error(), "not a Forst-managed link or directory") {
		t.Fatalf("error = %v", err)
	}
	got, readErr := os.ReadFile(linkPath)
	if readErr != nil {
		t.Fatalf("link path should still exist: %v", readErr)
	}
	if string(got) != "not a package" {
		t.Fatalf("regular file was modified: %q", got)
	}
}

func TestGenerateLink_treatsCorruptMarkerAsForeign(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	log := discardLinkLog()
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if err := os.MkdirAll(linkPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkPath, ".forst-generated"), []byte(`{"boundaryRoot": `), 0o644); err != nil {
		t.Fatal(err)
	}

	err := linkGeneratedClient(boundary, outDir, "@forst/gen", log)
	if err == nil {
		t.Fatal("expected error for directory with corrupt Forst marker")
	}
	if !strings.Contains(err.Error(), "real directory that Forst did not create") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateLink_refusesToReplaceRealDirectory(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	log := discardLinkLog()
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if err := os.MkdirAll(linkPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkPath, "package.json"), []byte(`{"name":"foreign"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	err := linkGeneratedClient(boundary, outDir, "@forst/gen", log)
	if err == nil {
		t.Fatal("expected error for foreign real directory")
	}
	if !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateLink_skippedWhenNoNodeModules(t *testing.T) {
	boundary := t.TempDir()
	outDir := filepath.Join(boundary, ".forst", "client")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	if err := linkGeneratedClient(boundary, outDir, "@forst/gen", log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "no node_modules") {
		t.Fatalf("expected skip warning, got %q", buf.String())
	}
}

func TestGenerateLink_skippedForYarnPnP(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	pnp := filepath.Join(filepath.Dir(nodeModules), ".pnp.cjs")
	if err := os.WriteFile(pnp, []byte("/* yarn pnp */"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	if err := linkGeneratedClient(boundary, outDir, "@forst/gen", log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Plug'n'Play") && !strings.Contains(buf.String(), "committed mode") {
		t.Fatalf("expected Yarn PnP skip warning, got %q", buf.String())
	}
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("link should not exist under PnP, err=%v", err)
	}
}

func TestGenerateLink_fallsBackToCopyWhenSymlinkDenied(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	if err := os.WriteFile(filepath.Join(outDir, "package.json"), []byte(`{"name":"@forst/gen"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	origSymlink := generateLinkIO.Symlink
	origJunction := generateLinkIO.Junction
	origGOOS := generateLinkIO.GOOS
	t.Cleanup(func() {
		generateLinkIO.Symlink = origSymlink
		generateLinkIO.Junction = origJunction
		generateLinkIO.GOOS = origGOOS
	})
	generateLinkIO.GOOS = "linux"
	generateLinkIO.Symlink = func(_, _ string) error {
		return fmt.Errorf("operation not permitted")
	}
	generateLinkIO.Junction = func(_, _ string) error {
		return fmt.Errorf("junction unused")
	}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	if err := linkGeneratedClient(boundary, outDir, "@forst/gen", log); err != nil {
		t.Fatalf("linkGeneratedClient: %v", err)
	}
	if !strings.Contains(buf.String(), "falling back to a recursive copy") {
		t.Fatalf("expected copy fallback warning, got %q", buf.String())
	}
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat copy dest: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected real directory copy, got symlink")
	}
	if !info.IsDir() {
		t.Fatal("expected directory at link path")
	}
	if _, err := os.Stat(filepath.Join(linkPath, "package.json")); err != nil {
		t.Fatalf("copied package.json missing: %v", err)
	}
}

func TestGenerateLink_failsWhenLinkBelongsToAnotherBoundary(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	log := discardLinkLog()
	packageName := "@forst/gen"

	otherBoundary := t.TempDir()
	otherOut := filepath.Join(otherBoundary, ".forst", "client")
	if err := os.MkdirAll(otherOut, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeForstGeneratedMarker(otherOut, otherBoundary, packageName, nil); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	linkFixtureTarget(t, otherOut, linkPath)

	err := linkGeneratedClient(boundary, outDir, packageName, log)
	if err == nil {
		t.Fatal("expected ownership conflict error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "already belongs to another Forst project") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(msg, otherBoundary) || !strings.Contains(msg, boundary) {
		t.Fatalf("error should name both boundaries, got %v", err)
	}
	if !strings.Contains(msg, "generate.packageName") {
		t.Fatalf("error should suggest packageName, got %v", err)
	}
}

func TestGenerateLink_markerRecordsBoundaryRootAndPackageName(t *testing.T) {
	boundary, outDir, _ := setupLinkFixture(t)
	log := discardLinkLog()
	packageName := "@forst/gen"

	if err := linkGeneratedClient(boundary, outDir, packageName, log); err != nil {
		t.Fatalf("linkGeneratedClient: %v", err)
	}
	marker, err := readForstGeneratedMarker(outDir)
	if err != nil {
		t.Fatalf("read marker: %v", err)
	}
	wantBoundary, err := filepath.Abs(boundary)
	if err != nil {
		t.Fatal(err)
	}
	if marker.BoundaryRoot != wantBoundary {
		t.Fatalf("boundaryRoot = %q, want %q", marker.BoundaryRoot, wantBoundary)
	}
	if marker.PackageName != packageName {
		t.Fatalf("packageName = %q, want %q", marker.PackageName, packageName)
	}

	raw, err := os.ReadFile(filepath.Join(outDir, forstGeneratedMarkerName))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["boundaryRoot"] != wantBoundary || decoded["packageName"] != packageName {
		t.Fatalf("marker json = %v", decoded)
	}
}

func TestGenerateLink_skippedWhenLinkNever(t *testing.T) {
	boundary, outDir, nodeModules := setupLinkFixture(t)
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	genCfg := ftconfig.GenerateConfig{
		PackageName: "@forst/gen",
		OutDir:      ".forst/client",
		Link:        "never",
		Emit:        "js",
	}
	if err := maybeLinkGeneratedClient(boundary, outDir, genCfg, log); err != nil {
		t.Fatalf("maybeLinkGeneratedClient: %v", err)
	}
	linkPath := filepath.Join(nodeModules, "@forst", "gen")
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("link should not exist when link=never, err=%v", err)
	}
	if !strings.Contains(buf.String(), "skipping node_modules link") {
		t.Fatalf("expected debug skip, got %q", buf.String())
	}
}

func setupLinkFixture(t *testing.T) (boundary, outDir, nodeModules string) {
	t.Helper()
	boundary = t.TempDir()
	outDir = filepath.Join(boundary, ".forst", "client")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nodeModules = filepath.Join(boundary, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		t.Fatal(err)
	}
	return boundary, outDir, nodeModules
}

func linkFixtureTarget(t *testing.T, target, linkPath string) {
	t.Helper()
	if err := createDirLink(target, linkPath, discardLinkLog()); err != nil {
		t.Fatalf("createDirLink(%q -> %q): %v", target, linkPath, err)
	}
}

func discardLinkLog() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(io.Discard)
	log.SetLevel(logrus.DebugLevel)
	return log
}
