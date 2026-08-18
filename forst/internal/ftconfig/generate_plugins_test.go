package ftconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGeneratePluginConfig_Validate(t *testing.T) {
	valid := GeneratePluginConfig{Name: "echo", Cmd: "forst-gen-echo", Out: "generated/echo"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if valid.EffectiveOutDir("/proj") != "/proj/generated/echo" {
		t.Fatalf("EffectiveOutDir")
	}

	cases := []GeneratePluginConfig{
		{Name: "", Cmd: "x", Out: "out"},
		{Name: "x", Cmd: "", Out: "out"},
		{Name: "x", Cmd: "x", Out: ""},
		{Name: "x", Cmd: "x", Out: "/abs"},
		{Name: "x", Cmd: "x", Out: "../escape"},
	}
	for _, c := range cases {
		if err := c.Validate(); err == nil {
			t.Fatalf("expected error for %#v", c)
		}
	}
}

func TestGeneratePluginConfig_ResolveCmd_relativePath(t *testing.T) {
	cfg := GeneratePluginConfig{Name: "echo", Cmd: "./bin/my-plugin", Out: "generated/echo"}
	got, err := cfg.ResolveCmd("/proj")
	if err != nil {
		t.Fatalf("ResolveCmd: %v", err)
	}
	want := filepath.Join("/proj", "bin", "my-plugin")
	if got != want {
		t.Fatalf("ResolveCmd = %q, want %q", got, want)
	}
}

func TestGeneratePluginConfig_ResolveCmd_absolutePath(t *testing.T) {
	cfg := GeneratePluginConfig{Name: "echo", Cmd: "/usr/local/bin/forst-gen-echo", Out: "generated/echo"}
	got, err := cfg.ResolveCmd("/proj")
	if err != nil {
		t.Fatalf("ResolveCmd: %v", err)
	}
	if got != "/usr/local/bin/forst-gen-echo" {
		t.Fatalf("ResolveCmd = %q", got)
	}
}

func TestGeneratePluginConfig_ResolveCmd_lookPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("LookPath executable semantics differ on Windows")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "forst-gen-test-cmd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	cfg := GeneratePluginConfig{Name: "echo", Cmd: "forst-gen-test-cmd", Out: "generated/echo"}
	got, err := cfg.ResolveCmd("/proj")
	if err != nil {
		t.Fatalf("ResolveCmd: %v", err)
	}
	if got != bin {
		t.Fatalf("ResolveCmd = %q, want %q", got, bin)
	}
}

func TestGenerateConfig_Validate_plugins(t *testing.T) {
	cfg := GenerateConfig{
		PackageName:    DefaultPackageName,
		OutDir:         ".forst/client",
		Link:           "auto",
		Emit:           "js",
		TestingSubpath: "$testing",
		Plugins: []GeneratePluginConfig{
			{Name: "bad", Cmd: "x", Out: "../nope"},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected plugin validation error")
	}
}
