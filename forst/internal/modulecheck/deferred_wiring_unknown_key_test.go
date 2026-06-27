package modulecheck_test

import (
	"path/filepath"
	"testing"

	"forst/internal/modulecheck"
)

func TestCheckModuleProviders_unknownWiringKeyRejectedAfterMerge(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module unk_after_merge\n\ngo 1.23\n")
	writeFile(t, filepath.Join(dir, "demo.ft"), `package main

import "testing"

type Logger = { info(msg String) }

func expireToken() {
	use logger: Logger
}

func TestX(t *testing.T) {
	with { BadKey: 1 } {
		expireToken()
	}
}
`)
	_, err := modulecheck.CheckModuleProviders(nil, modulecheck.Options{ModuleRoot: dir})
	if err == nil {
		t.Fatal("expected unknown wiring key error after module merge")
	}
}
