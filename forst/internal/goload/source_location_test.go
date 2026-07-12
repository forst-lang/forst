package goload

import (
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestFirstNonTestGoFile_skipsTests(t *testing.T) {
	t.Parallel()
	pkg := &packages.Package{
		GoFiles: []string{"/tmp/a_test.go", "/tmp/a.go"},
	}
	if got := FirstNonTestGoFile(pkg); got != "/tmp/a.go" {
		t.Fatalf("got %q", got)
	}
}

func TestPackageFileLocation_returnsFirstFile(t *testing.T) {
	t.Parallel()
	pkg := &packages.Package{GoFiles: []string{"/proj/helpers.go"}}
	loc, ok := PackageFileLocation(pkg)
	if !ok || loc.File != "/proj/helpers.go" || loc.IsSet() {
		t.Fatalf("loc=%+v ok=%v", loc, ok)
	}
}
