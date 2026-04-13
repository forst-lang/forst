package typechecker

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/forstpkg"

	"github.com/sirupsen/logrus"
)

func TestCheckTypes_mergedPackage_crossFileResultOkNarrowing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	resultPath := filepath.Join(dir, "result.ft")
	mainPath := filepath.Join(dir, "main.ft")

	if err := os.WriteFile(resultPath, []byte(`package demo

func parse(n Int): Result(Int, Error) {
	return n
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(`package demo

func use(n Int): Int {
	value := parse(n)
	if value is Ok() {
		return value
	}
	return 0
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	tc := mergedPackageTypecheck(t, []string{resultPath, mainPath})
	if tc == nil {
		t.Fatal("expected non-nil typechecker")
	}
}

func TestCheckTypes_mergedPackage_crossFileTypeGuardUse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	guardPath := filepath.Join(dir, "guard.ft")
	mainPath := filepath.Join(dir, "main.ft")

	if err := os.WriteFile(guardPath, []byte(`package demo

type Password = String

is (password Password) Strong {
	ensure password is Min(8)
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte(`package demo

func use(password Password): Password {
	if password is Strong() {
		return password
	}
	return password
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	tc := mergedPackageTypecheck(t, []string{guardPath, mainPath})
	if tc == nil {
		t.Fatal("expected non-nil typechecker")
	}
}

func mergedPackageTypecheck(t *testing.T, files []string) *TypeChecker {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)

	merged, _, err := forstpkg.ParseAndMergePackage(log, files)
	if err != nil {
		t.Fatalf("merge package: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(merged); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	return tc
}
