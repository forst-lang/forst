package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveGenerateGoPlan_cliOverridesMergeWithFtconfig(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Generate.Go.Entry = "./other.ft"
	cfg.Generate.Go.Out = "./out/main.go"
	opts := generateOptions{
		target:      dir,
		targetIsDir: true,
		goEntry:     entry,
	}
	plan, err := resolveGenerateGoPlan(opts, cfg, dir)
	if err != nil {
		t.Fatalf("resolveGenerateGoPlan: %v", err)
	}
	if !plan.active || plan.entryPath != entry {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.outPath != filepath.Join(dir, "out", "main.go") {
		t.Fatalf("outPath = %q", plan.outPath)
	}
}

func TestResolveGenerateGoPlan_inactiveWhenUnset(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	opts := generateOptions{target: dir, targetIsDir: true}
	plan, err := resolveGenerateGoPlan(opts, cfg, dir)
	if err != nil {
		t.Fatalf("resolveGenerateGoPlan: %v", err)
	}
	if plan.active {
		t.Fatalf("expected inactive plan, got %+v", plan)
	}
}

func TestRunGenerateGoSources_writesGoFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{
  "server": {"embedded": true},
  "files": {"include": ["**/*.ft"]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "main.ft")
	src := `package main

func Echo(input { message: String }) {
	return { echo: input.message }
}

func main() {
	println("ok")
}
`
	if err := os.WriteFile(entry, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "out", "main.go")
	cfg := DefaultConfig()
	plan := generateGoPlan{
		active:    true,
		entryPath: entry,
		outPath:   outPath,
	}
	log := newGenerateLogger()
	if err := runGenerateGoSources(plan, cfg, log); err != nil {
		t.Fatalf("runGenerateGoSources: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("missing output: %v", err)
	}
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "package main") {
		t.Fatalf("output = %q", body)
	}
}

func TestShouldSkipClientGenerate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Generate.SkipClient = true
	if !shouldSkipClientGenerate(generateOptions{}, cfg) {
		t.Fatal("expected skip from ftconfig")
	}
	if !shouldSkipClientGenerate(generateOptions{skipClient: true}, DefaultConfig()) {
		t.Fatal("expected skip from CLI flag")
	}
}

func TestRunMain_buildRejectsGoOutputPath(t *testing.T) {
	dir := t.TempDir()
	ft := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(ft, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code := runMain([]string{"forst", "build", "-o", filepath.Join(dir, "out.go"), ft})
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
}

func TestResolveGenerateGoPlan_relativeConfigFromOutsideBoundary(t *testing.T) {
	boundary := t.TempDir()
	outside := t.TempDir()
	entry := filepath.Join(boundary, "main.ft")
	if err := os.WriteFile(entry, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Generate.Go.Entry = "./main.ft"
	cfg.Generate.Go.Out = "./out/main.go"
	opts := generateOptions{target: boundary, targetIsDir: true}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(outside); err != nil {
		t.Fatal(err)
	}

	plan, err := resolveGenerateGoPlan(opts, cfg, boundary)
	if err != nil {
		t.Fatalf("resolveGenerateGoPlan: %v", err)
	}
	wantEntry := filepath.Join(boundary, "main.ft")
	wantOut := filepath.Join(boundary, "out", "main.go")
	if plan.entryPath != wantEntry {
		t.Fatalf("entryPath = %q want %q", plan.entryPath, wantEntry)
	}
	if plan.outPath != wantOut {
		t.Fatalf("outPath = %q want %q", plan.outPath, wantOut)
	}
}
