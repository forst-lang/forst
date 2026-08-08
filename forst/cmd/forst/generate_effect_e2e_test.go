package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate_effectMode_tscFixtureAndRuntime(t *testing.T) {
	if os.Getenv("FORST_SKIP_EFFECT_E2E") == "1" {
		t.Skip("FORST_SKIP_EFFECT_E2E=1")
	}
	if os.Getenv("FORST_SKIP_TS_E2E") == "1" {
		t.Skip("FORST_SKIP_TS_E2E=1")
	}

	repoEffect := findRepoEffectModule()
	if repoEffect == "" {
		t.Skip("repo effect package not found (bun install at monorepo root)")
	}

	dir := t.TempDir()
	writeMainFt(t, dir, generateTestMinimalValidForst)
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"generate":{"effect":true,"link":"always"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repoEffect, filepath.Join(dir, "node_modules", "effect")); err != nil {
		t.Fatalf("symlink effect: %v", err)
	}

	if err := generateCommand([]string{dir}); err != nil {
		t.Fatalf("generateCommand: %v", err)
	}

	consumer, err := os.ReadFile(filepath.Join("testdata", "effect", "consumer.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "consumer.ts"), consumer, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "types": []
  },
  "include": ["consumer.ts", ".forst/client/dist/**/*.d.ts"]
}`
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runTsc(t, dir); err != nil {
		t.Fatalf("effect tsc fixture failed: %v", err)
	}

	strictCfg := `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "types": []
  },
  "include": ["` + clientDistIncludeGlob(dir) + `"]
}`
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.strict-dts.json"), []byte(strictCfg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runTscConfig(t, dir, filepath.Join(dir, "tsconfig.strict-dts.json")); err != nil {
		t.Fatalf("strict generated .d.ts tsc failed: %v", err)
	}

	runtime, err := os.ReadFile(filepath.Join("testdata", "effect", "runtime.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(dir, "runtime.mjs")
	if err := os.WriteFile(runtimePath, runtime, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runNodeRequireSmoke(t, dir, runtimePath)
	if err != nil {
		t.Fatalf("effect runtime fixture failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "effect-runtime-ok") {
		t.Fatalf("unexpected runtime output:\n%s", out)
	}
}

func findRepoEffectModule() string {
	root := findMonorepoRootWithPackageJSON()
	if root == "" {
		return ""
	}
	for _, base := range []string{root, filepath.Dir(root)} {
		cand := filepath.Join(base, "node_modules", "effect")
		if st, err := os.Stat(filepath.Join(cand, "package.json")); err == nil && !st.IsDir() {
			abs, absErr := filepath.Abs(cand)
			if absErr != nil {
				return cand
			}
			return abs
		}
	}
	return ""
}
