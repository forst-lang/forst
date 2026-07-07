package testutil

import (
	"errors"
	"os"
	"testing"
)

func TestWriteMixedGoForstModule(t *testing.T) {
	root, importPath := WriteMixedGoForstModule(t, "mixed")
	if root == "" || importPath != "mixedtest/mixed" {
		t.Fatalf("got root=%q importPath=%q", root, importPath)
	}
}

func TestWriteMixedGoForstModule_defaultModule(t *testing.T) {
	root, importPath := WriteMixedGoForstModule(t, "")
	if root == "" || importPath != "mixedtest/mixed" {
		t.Fatalf("got root=%q importPath=%q", root, importPath)
	}
}

func TestWriteMixedGoForstModule_mkdirError(t *testing.T) {
	orig := mixedGoMkdirAll
	t.Cleanup(func() { mixedGoMkdirAll = orig })
	mixedGoMkdirAll = func(string, os.FileMode) error {
		return errors.New("mkdir failed")
	}
	msg := stubFail(t, func() {
		WriteMixedGoForstModule(t, "mixed")
	})
	if msg != "mkdir failed" {
		t.Fatalf("msg = %q", msg)
	}
}

func TestWriteMixedGoForstModule_writeError(t *testing.T) {
	orig := mixedGoWriteFile
	t.Cleanup(func() { mixedGoWriteFile = orig })
	mixedGoWriteFile = func(string, []byte, os.FileMode) error {
		return errors.New("write failed")
	}
	msg := stubFail(t, func() {
		WriteMixedGoForstModule(t, "mixed")
	})
	if msg != "write failed" {
		t.Fatalf("msg = %q", msg)
	}
}
