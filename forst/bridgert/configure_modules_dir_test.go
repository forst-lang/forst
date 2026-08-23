package bridgert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
)

func TestValidateCompiledModulesDir_missing(t *testing.T) {
	err := validateCompiledModulesDir(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
	if got := err.Error(); !strings.Contains(got, "does not exist") || !strings.Contains(got, ftconfig.EnvBridgeModulesDir) {
		t.Fatalf("err = %q", got)
	}
}

func TestValidateCompiledModulesDir_notDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateCompiledModulesDir(file)
	if err == nil {
		t.Fatal("expected error for file path")
	}
}
