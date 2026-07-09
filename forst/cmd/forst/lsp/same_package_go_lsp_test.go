package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/goload"
	"forst/internal/testmod"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

const samePkgGoFtSource = `package memos

import "fmt"
import "os"

func main() {
  data, readErr := os.ReadFile("/tmp/sample.txt")
  if readErr == nil {
    fmt.Print(StringFromBytes(data))
  }
}
`

func writeSamePackageGoLSPFixture(t *testing.T) (ftPath, uri string) {
	t.Helper()
	root, _ := testutil.WriteMixedGoForstModule(t, "memos")
	ftPath = filepath.Join(root, "memos", "main.ft")
	if err := os.WriteFile(ftPath, []byte(samePkgGoFtSource), 0o644); err != nil {
		t.Fatal(err)
	}
	return ftPath, mustFileURI(t, ftPath)
}

func assertNoUnknownIdentifier(t *testing.T, diags []LSPDiagnostic, name string) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, "unknown identifier") && strings.Contains(d.Message, name) {
			t.Fatalf("unexpected unknown identifier for %s: %#v", name, d)
		}
	}
}

func TestCompileForstFile_samePackageGoCall_resolvesExportedFunc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	ftPath, _ := writeSamePackageGoLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	diags := s.compileForstFile(ftPath, samePkgGoFtSource, nil)
	assertNoUnknownIdentifier(t, diags, "StringFromBytes")
}

func TestProcessForstFile_samePackageGoCall_resolvesExportedFunc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	_, uri := writeSamePackageGoLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = samePkgGoFtSource
	s.documentMu.Unlock()

	diags := s.processForstFile(uri, samePkgGoFtSource)
	assertNoUnknownIdentifier(t, diags, "StringFromBytes")
}

func TestAnalyzeForstDocument_samePackageGoCall_resolvesExportedFunc(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	_, uri := writeSamePackageGoLSPFixture(t)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = samePkgGoFtSource
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyzeForstDocument failed")
	}
	if ctx.CheckErr != nil && strings.Contains(ctx.CheckErr.Error(), "StringFromBytes") {
		t.Fatalf("typecheck: %v", ctx.CheckErr)
	}
	if ctx.TC == nil || !ctx.TC.SamePackageGoLoaded() {
		t.Fatal("expected same-package Go loaded in analyzeForstDocument")
	}
}

func TestBuildPackageSnapshot_samePackageGoLoaded_withMergedPeers(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	root, importPath := testutil.WriteMixedGoForstModule(t, "memos")
	pkgDir := filepath.Join(root, "memos")
	mainPath := filepath.Join(pkgDir, "main.ft")
	peerPath := filepath.Join(pkgDir, "peer.ft")
	if err := os.WriteFile(mainPath, []byte(samePkgGoFtSource), 0o644); err != nil {
		t.Fatal(err)
	}
	const peerSrc = `package memos

func peerMarker(): Int {
  return 1
}
`
	if err := os.WriteFile(peerPath, []byte(peerSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewLSPServer("8080", logrus.New())
	uriMain := mustFileURI(t, mainPath)
	uriPeer := mustFileURI(t, peerPath)
	uris := []string{uriMain, uriPeer}
	contents := map[string]string{uriMain: samePkgGoFtSource, uriPeer: peerSrc}

	results, err := s.parsePackageGroupMembersParallel(uris, contents)
	if err != nil {
		t.Fatal(err)
	}
	for i := range results {
		if results[i].ParseErr != nil {
			t.Fatalf("parse err file %d: %v", i, results[i].ParseErr)
		}
	}

	snap := s.buildPackageSnapshot(uris, results, pkgDir)
	if snap == nil || snap.tc == nil {
		t.Fatal("expected non-nil snapshot with typechecker")
	}
	if !snap.tc.SamePackageGoLoaded() {
		t.Fatalf("expected SamePackageGoLoaded for import path %q", importPath)
	}
	if snap.checkErr != nil && strings.Contains(snap.checkErr.Error(), "StringFromBytes") {
		t.Fatalf("merged typecheck: %v", snap.checkErr)
	}
}

func TestBuildPackageSnapshot_samePackageGoLoaded_moduleRootFromNestedDir(t *testing.T) {
	t.Parallel()
	goload.ClearLoadCacheForTest()
	root := t.TempDir()
	testmod.WriteGoMod(t, root, "nestedlsp")
	pkgDir := filepath.Join(root, "internal", "memos")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const goSrc = `package memos

func StringFromBytes(b []byte) string {
	return string(b)
}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "text.go"), []byte(goSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	ftPath := filepath.Join(pkgDir, "run.ft")
	if err := os.WriteFile(ftPath, []byte(samePkgGoFtSource), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewLSPServer("8080", logrus.New())
	uri := mustFileURI(t, ftPath)
	results, err := s.parsePackageGroupMembersParallel([]string{uri}, map[string]string{uri: samePkgGoFtSource})
	if err != nil {
		t.Fatal(err)
	}

	snap := s.buildPackageSnapshot([]string{uri}, results, pkgDir)
	if snap == nil || snap.tc == nil {
		t.Fatal("expected snapshot")
	}
	if !snap.tc.SamePackageGoLoaded() {
		t.Fatal("expected SamePackageGoLoaded for nested package dir")
	}
}
