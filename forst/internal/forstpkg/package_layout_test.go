package forstpkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateGoPackageLayout_emptyMap(t *testing.T) {
	if err := ValidateGoPackageLayout(nil); err != nil {
		t.Fatalf("expected nil for empty map, got %v", err)
	}
	if err := ValidateGoPackageLayout(map[string][]string{}); err != nil {
		t.Fatalf("expected nil for empty map, got %v", err)
	}
}

func TestValidateOneDirectoryPerPackage_emptyMap(t *testing.T) {
	if err := ValidateOneDirectoryPerPackage(nil); err != nil {
		t.Fatalf("expected nil for empty map, got %v", err)
	}
}

func TestValidateOneDirectoryPerPackage_singleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "auth.ft")
	err := ValidateOneDirectoryPerPackage(map[string][]string{
		"auth": {file},
	})
	if err != nil {
		t.Fatalf("expected nil for single file, got %v", err)
	}
}

func TestValidateOneDirectoryPerPackage_sameDirMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	err := ValidateOneDirectoryPerPackage(map[string][]string{
		"auth": {
			filepath.Join(dir, "auth.ft"),
			filepath.Join(dir, "bcrypt.ft"),
		},
	})
	if err != nil {
		t.Fatalf("expected nil for same-dir multi-file package, got %v", err)
	}
}

func TestValidateOneDirectoryPerPackage_multiDirSamePackage(t *testing.T) {
	root := t.TempDir()
	authDir := filepath.Join(root, "auth")
	cryptoDir := filepath.Join(root, "auth", "crypto")
	if err := mkdirAll(authDir, cryptoDir); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(authDir, "a.ft")
	b := filepath.Join(cryptoDir, "b.ft")
	err := ValidateGoPackageLayout(map[string][]string{
		"auth": {a, b},
	})
	if err == nil {
		t.Fatal("expected error for same package in sibling directories")
	}
	msg := err.Error()
	if !strings.Contains(msg, `package "auth" spans 2 directories`) {
		t.Fatalf("expected package span message, got %q", msg)
	}
}

func TestValidateGoPackageLayout_differentPackagesSameDir(t *testing.T) {
	dir := t.TempDir()
	err := ValidateGoPackageLayout(map[string][]string{
		"auth": {filepath.Join(dir, "auth.ft")},
		"main": {filepath.Join(dir, "main.ft")},
	})
	if err == nil {
		t.Fatal("expected error when different packages share a directory")
	}
	if !strings.Contains(err.Error(), "contains 2 packages") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGoPackageLayout_externalTestPackageAllowed(t *testing.T) {
	dir := t.TempDir()
	err := ValidateGoPackageLayout(map[string][]string{
		"auth":      {filepath.Join(dir, "auth.ft")},
		"auth_test": {filepath.Join(dir, "auth_test.ft")},
	})
	if err != nil {
		t.Fatalf("expected nil for auth + auth_test in same directory, got %v", err)
	}
}

func TestValidateGoPackageLayout_validMultiPackageTree(t *testing.T) {
	root := t.TempDir()
	err := ValidateGoPackageLayout(map[string][]string{
		"main": {filepath.Join(root, "main", "main.ft")},
		"auth": {
			filepath.Join(root, "auth", "auth.ft"),
			filepath.Join(root, "auth", "bcrypt.ft"),
		},
	})
	if err != nil {
		t.Fatalf("expected nil for Go-valid tree, got %v", err)
	}
}

func mkdirAll(dirs ...string) error {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
