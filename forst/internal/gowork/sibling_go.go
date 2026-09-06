package gowork

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyHandwrittenGoSources copies same-package hand-written .go files from srcDir to dstDir
// so sandboxes (forst run, forst test, etc.) can link mixed Forst+Go packages.
// Skips Forst-generated *.gen.go and any z_forst_* emit leftovers.
func CopyHandwrittenGoSources(srcDir, dstDir string) error {
	if srcDir == "" || dstDir == "" {
		return nil
	}
	srcDir = filepath.Clean(srcDir)
	dstDir = filepath.Clean(dstDir)
	if srcDir == dstDir {
		return nil
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, ".gen.go") || strings.HasPrefix(name, "z_forst_") {
			continue
		}
		srcPath := filepath.Join(srcDir, name)
		dstPath := filepath.Join(dstDir, name)
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}
