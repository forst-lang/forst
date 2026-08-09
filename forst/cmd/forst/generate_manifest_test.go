package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate_listJSONManifest(t *testing.T) {
	dir := t.TempDir()
	writeAcceptanceBcryptAuth(t, dir)

	var buf bytes.Buffer
	prev := generateReportWriter
	generateReportWriter = &buf
	t.Cleanup(func() { generateReportWriter = prev })

	if err := generateCommand([]string{"--list", dir}); err != nil {
		t.Fatalf("generateCommand --list: %v", err)
	}

	var manifest generateManifest
	if err := json.Unmarshal(buf.Bytes(), &manifest); err != nil {
		t.Fatalf("manifest JSON: %v\n%s", err, buf.String())
	}
	if manifest.PackageName != "@forst/gen" {
		t.Fatalf("packageName = %q, want @forst/gen", manifest.PackageName)
	}
	if len(manifest.Packages) < 2 {
		t.Fatalf("packages = %v, want bcrypt and auth", manifest.Packages)
	}
	foundCompare := false
	for _, fn := range manifest.Functions {
		if fn.Package == "bcrypt" && fn.Function == "ComparePassword" {
			foundCompare = true
		}
	}
	if !foundCompare {
		t.Fatalf("ComparePassword missing from functions: %+v", manifest.Functions)
	}
	if _, err := os.Stat(filepath.Join(defaultClientOutDir(dir), "package.json")); err == nil {
		t.Fatal("--list must not write client output")
	}
}
