package ftconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolvePluginCmd_relativeAndAbsolute(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "custom-plugin")
	if err := os.WriteFile(abs, []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePluginCmd("/proj", abs)
	if err != nil || got != abs {
		t.Fatalf("absolute: got %q err %v", got, err)
	}
	got, err = resolvePluginCmd("/proj", "./bin/forst-gen-echo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/proj", "bin", "forst-gen-echo")
	if got != want {
		t.Fatalf("relative = %q want %q", got, want)
	}
}

func TestResolvePluginCmd_pluginDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on Windows")
	}
	dir := t.TempDir()
	name := "forst-gen-echo"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvForstPluginDir, dir)
	got, err := resolvePluginCmd("/proj", name)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("got %q want %q", got, path)
	}
}

func TestResolvePluginCmd_missingBareName(t *testing.T) {
	t.Setenv(EnvForstPluginDir, "")
	t.Setenv(EnvForstPluginsPath, "")
	_, err := resolvePluginCmd("/proj", "forst-gen-definitely-missing")
	if err == nil {
		t.Fatal("expected error")
	}
}
