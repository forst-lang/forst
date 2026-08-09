package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestGenerate_warnsWhenNoLifecycleScriptRunsGenerate(t *testing.T) {
	boundary := t.TempDir()
	if err := os.WriteFile(filepath.Join(boundary, "package.json"), []byte(`{
  "name": "app",
  "scripts": { "dev": "vite" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	genCfg := ftconfig.GenerateConfig{
		PackageName: "@forst/gen",
		OutDir:      ".forst/client",
		Link:        "auto",
		Emit:        "js",
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	warnMissingLifecycleScript(boundary, genCfg, log)
	got := buf.String()
	if !strings.Contains(got, "no lifecycle script runs forst generate") {
		t.Fatalf("expected warning, got %q", got)
	}
	if !strings.Contains(got, "postinstall") || !strings.Contains(got, "forst generate .") {
		t.Fatalf("expected pasteable postinstall JSON, got %q", got)
	}
	if !strings.Contains(got, `"scripts"`) && !strings.Contains(got, `\"scripts\"`) {
		t.Fatalf("expected scripts key in pasteable JSON, got %q", got)
	}
	if !strings.Contains(got, "@forst/gen") {
		t.Fatalf("expected package name in warning, got %q", got)
	}
}

func TestGenerate_doesNotWarnWhenPostinstallRunsGenerate(t *testing.T) {
	boundary := t.TempDir()
	if err := os.WriteFile(filepath.Join(boundary, "package.json"), []byte(`{
  "name": "app",
  "scripts": { "postinstall": "forst generate ." }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	genCfg := ftconfig.GenerateConfig{
		PackageName: "@forst/gen",
		OutDir:      ".forst/client",
		Link:        "auto",
		Emit:        "js",
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	warnMissingLifecycleScript(boundary, genCfg, log)
	if buf.Len() != 0 {
		t.Fatalf("expected no warning, got %q", buf.String())
	}
}

func TestGenerate_doesNotWarnInCommittedMode(t *testing.T) {
	boundary := t.TempDir()
	if err := os.WriteFile(filepath.Join(boundary, "package.json"), []byte(`{
  "name": "app",
  "scripts": { "dev": "vite" }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	genCfg := ftconfig.GenerateConfig{
		PackageName: "@forst/gen",
		OutDir:      "src/forst-client",
		Link:        "never",
		Emit:        "js",
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	warnMissingLifecycleScript(boundary, genCfg, log)
	if buf.Len() != 0 {
		t.Fatalf("expected no warning in committed mode, got %q", buf.String())
	}
}
