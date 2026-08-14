package ftconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultGenerateConfig_hasExpectedDefaults(t *testing.T) {
	g := Default().Generate
	if g.OutDir != ".forst/client" {
		t.Fatalf("OutDir: got %q, want .forst/client", g.OutDir)
	}
	if g.Link != "auto" {
		t.Fatalf("Link: got %q, want auto", g.Link)
	}
	if g.Emit != "js" {
		t.Fatalf("Emit: got %q, want js", g.Emit)
	}
	if g.TestingSubpath != "$testing" {
		t.Fatalf("TestingSubpath: got %q, want $testing", g.TestingSubpath)
	}
	if g.SSRModule != "" {
		t.Fatalf("SSRModule: got %q, want empty", g.SSRModule)
	}
}

func TestDefaultGenerateConfig_packageNameIsFixedForstGen(t *testing.T) {
	g := Default().Generate
	if g.PackageName != DefaultPackageName {
		t.Fatalf("PackageName: got %q, want %q", g.PackageName, DefaultPackageName)
	}
	if DefaultPackageName != "@forst/gen" {
		t.Fatalf("DefaultPackageName: got %q, want @forst/gen", DefaultPackageName)
	}
}

func TestDefaultGenerateConfig_effectDefaultsToFalse(t *testing.T) {
	if Default().Generate.Effect {
		t.Fatal("Effect should default to false")
	}
}

func TestDefaultGenerateConfig_omitStubsDefaultsToFalse(t *testing.T) {
	if Default().Generate.OmitStubs {
		t.Fatal("OmitStubs should default to false")
	}
}

func TestEffectiveGenerateConfig_packageNameIgnoresAdopterPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{"name":"@acme/web","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, configFileName)
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	g := EffectiveGenerateConfig(cfg, dir)
	if g.PackageName != DefaultPackageName {
		t.Fatalf("PackageName must ignore adopter package.json: got %q, want %q", g.PackageName, DefaultPackageName)
	}
}

func TestEffectiveGenerateConfig_overridesFromJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, configFileName)
	json := `{
  "generate": {
    "packageName": "@acme/api-client",
    "outDir": "packages/forst-client",
    "link": "never",
    "emit": "js",
    "testingSubpath": "$test-double",
    "effect": true,
    "ssrModule": "src/ssr.ts",
    "omitStubs": true
  }
}`
	if err := os.WriteFile(cfgPath, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	g := EffectiveGenerateConfig(cfg, dir)
	if g.PackageName != "@acme/api-client" {
		t.Fatalf("PackageName: %q", g.PackageName)
	}
	if g.OutDir != "packages/forst-client" {
		t.Fatalf("OutDir: %q", g.OutDir)
	}
	if g.Link != "never" {
		t.Fatalf("Link: %q", g.Link)
	}
	if g.Emit != "js" {
		t.Fatalf("Emit: %q", g.Emit)
	}
	if g.TestingSubpath != "$test-double" {
		t.Fatalf("TestingSubpath: %q", g.TestingSubpath)
	}
	if !g.Effect {
		t.Fatal("Effect: want true")
	}
	if g.SSRModule != "src/ssr.ts" {
		t.Fatalf("SSRModule: %q", g.SSRModule)
	}
	if !g.OmitStubs {
		t.Fatal("OmitStubs: want true")
	}
}

func TestGenerateConfig_Validate_rejectsEmitTs(t *testing.T) {
	g := Default().Generate
	g.Emit = "ts"
	err := g.Validate()
	if err == nil {
		t.Fatal("expected error for emit=ts")
	}
	if !strings.Contains(err.Error(), "emit") {
		t.Fatalf("error should mention emit: %v", err)
	}
}

func TestGenerateConfig_Validate_rejectsInvalidLinkMode(t *testing.T) {
	g := Default().Generate
	g.Link = "sometimes"
	err := g.Validate()
	if err == nil {
		t.Fatal("expected error for invalid link")
	}
	if !strings.Contains(err.Error(), "link") {
		t.Fatalf("error should mention link: %v", err)
	}
}

func TestGenerateConfig_Validate_rejectsOutDirEscapingBoundary(t *testing.T) {
	cases := []string{
		"../outside",
		"/abs/path",
		"foo/../../escape",
	}
	for _, outDir := range cases {
		t.Run(outDir, func(t *testing.T) {
			g := Default().Generate
			g.OutDir = outDir
			err := g.Validate()
			if err == nil {
				t.Fatalf("expected error for outDir %q", outDir)
			}
			if !strings.Contains(strings.ToLower(err.Error()), "outdir") &&
				!strings.Contains(err.Error(), "outDir") {
				t.Fatalf("error should mention outDir: %v", err)
			}
		})
	}
}

func TestGenerateConfig_Validate_rejectsInvalidPackageName(t *testing.T) {
	cases := []string{
		"",
		"Invalid Name",
		"@scope",
		"/leading-slash",
		"UPPERCASE",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			g := Default().Generate
			g.PackageName = name
			err := g.Validate()
			if err == nil {
				t.Fatalf("expected error for packageName %q", name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), "packagename") &&
				!strings.Contains(err.Error(), "packageName") {
				t.Fatalf("error should mention packageName: %v", err)
			}
		})
	}
}

func TestGenerateConfig_Validate_rejectsMultiSegmentTestingSubpath(t *testing.T) {
	g := Default().Generate
	g.TestingSubpath = "foo/bar"
	err := g.Validate()
	if err == nil {
		t.Fatal("expected error for multi-segment testingSubpath")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "testingsubpath") &&
		!strings.Contains(err.Error(), "testingSubpath") {
		t.Fatalf("error should mention testingSubpath: %v", err)
	}
}

func TestEffectiveOutDir_resolvesRelativeToBoundaryRoot(t *testing.T) {
	boundary := filepath.Join(t.TempDir(), "proj")
	g := Default().Generate
	got := g.EffectiveOutDir(boundary)
	want := filepath.Join(boundary, ".forst", "client")
	if got != want {
		t.Fatalf("default EffectiveOutDir: got %q, want %q", got, want)
	}

	g.OutDir = "packages/forst-client"
	got = g.EffectiveOutDir(boundary)
	want = filepath.Join(boundary, "packages", "forst-client")
	if got != want {
		t.Fatalf("custom EffectiveOutDir: got %q, want %q", got, want)
	}
}

func TestIsEphemeral_trueForDefaultOutDir(t *testing.T) {
	boundary := t.TempDir()
	g := Default().Generate
	if !g.IsEphemeral(boundary) {
		t.Fatal("default outDir under .forst should be ephemeral")
	}
}

func TestIsEphemeral_falseForWorkspacePackageOutDir(t *testing.T) {
	boundary := t.TempDir()
	g := Default().Generate
	g.OutDir = "packages/forst-client"
	if g.IsEphemeral(boundary) {
		t.Fatal("workspace package outDir should not be ephemeral")
	}
}

func TestShouldLink_autoLinksOnlyWhenEphemeral(t *testing.T) {
	boundary := t.TempDir()
	ephemeral := Default().Generate
	ephemeral.Link = "auto"
	if !ephemeral.ShouldLink(boundary) {
		t.Fatal("auto + ephemeral should link")
	}

	committed := Default().Generate
	committed.Link = "auto"
	committed.OutDir = "packages/forst-client"
	if committed.ShouldLink(boundary) {
		t.Fatal("auto + non-ephemeral should not link")
	}

	always := committed
	always.Link = "always"
	if !always.ShouldLink(boundary) {
		t.Fatal("always should link even when non-ephemeral")
	}
}

func TestShouldLink_neverDisablesLinking(t *testing.T) {
	boundary := t.TempDir()
	g := Default().Generate
	g.Link = "never"
	if g.ShouldLink(boundary) {
		t.Fatal("never should disable linking even for ephemeral outDir")
	}
}

func TestValidate_rejectsReservedErrorsPackageName(t *testing.T) {
	g := Default().Generate
	g.PackageName = ReservedErrorsPackageName
	err := g.Validate()
	if err == nil {
		t.Fatal("expected error for reserved @forst/errors package name")
	}
	if !strings.Contains(err.Error(), ReservedErrorsPackageName) {
		t.Fatalf("error = %v", err)
	}
}

func TestReservedSubpaths_followsTestingSubpathConfig(t *testing.T) {
	g := Default().Generate
	reserved := g.ReservedSubpaths()
	if _, ok := reserved["$testing"]; !ok {
		t.Fatalf("default reserved map missing $testing: %#v", reserved)
	}

	g.TestingSubpath = "$test-double"
	reserved = g.ReservedSubpaths()
	if _, ok := reserved["$test-double"]; !ok {
		t.Fatalf("reserved map should follow testingSubpath: %#v", reserved)
	}
	if _, ok := reserved["$testing"]; ok {
		t.Fatalf("old $testing key should not remain: %#v", reserved)
	}
}

func TestReservedSubpaths_includesEffectWhenEnabled(t *testing.T) {
	g := Default().Generate
	g.Effect = true
	reserved := g.ReservedSubpaths()
	if _, ok := reserved["$testing"]; !ok {
		t.Fatalf("effect mode must reserve $testing: %#v", reserved)
	}
	if _, ok := reserved["$transport"]; !ok {
		t.Fatalf("effect mode must reserve $transport: %#v", reserved)
	}
	if _, ok := reserved["$errors"]; !ok {
		t.Fatalf("effect mode must reserve $errors: %#v", reserved)
	}
	if _, ok := reserved["$effect"]; ok {
		t.Fatalf("$effect should not be reserved: %#v", reserved)
	}
}

func TestGenerateConfig_Validate_rejectsTestingSubpathErrorsWithoutEffect(t *testing.T) {
	g := Default().Generate
	g.Effect = false
	g.TestingSubpath = "$errors"
	err := g.Validate()
	if err == nil {
		t.Fatal("expected validation error for testingSubpath $errors")
	}
	if !strings.Contains(err.Error(), "$errors") {
		t.Fatalf("error must mention $errors conflict: %v", err)
	}
}

func TestGenerateConfig_Validate_rejectsTestingSubpathTransport(t *testing.T) {
	g := Default().Generate
	g.Effect = false
	g.TestingSubpath = "$transport"
	err := g.Validate()
	if err == nil {
		t.Fatal("expected validation error for testingSubpath $transport")
	}
	if !strings.Contains(err.Error(), "$transport") {
		t.Fatalf("error must mention $transport conflict: %v", err)
	}
}

func TestGenerateConfig_Validate_rejectsEffectWithTestingSubpathEffect(t *testing.T) {
	g := Default().Generate
	g.Effect = true
	g.TestingSubpath = "$effect"
	err := g.Validate()
	if err == nil {
		t.Fatal("expected validation error for testingSubpath $effect with generate.effect")
	}
	if !strings.Contains(err.Error(), "$effect") {
		t.Fatalf("error must mention $effect conflict: %v", err)
	}
}
