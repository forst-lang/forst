package testrunner

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/gowork"
)

// copyHandwrittenGoSources copies same-package hand-written .go files into dstDir
// so forst test sandboxes can link mixed Forst+Go packages (tip handoff 03).
// Skips Forst-generated *.gen.go and any z_forst_* emit leftovers.
func copyHandwrittenGoSources(srcDir, dstDir string) error {
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

// collectGoOnlyPackageReplaces maps same-module Go-only packages (dirs with .go, no .ft)
// to their real directories so sandboxed go test can import them.
func collectGoOnlyPackageReplaces(moduleRoot, modulePath string, skipDirs map[string]struct{}) ([]gowork.PackageReplace, error) {
	var out []gowork.PackageReplace
	if moduleRoot == "" || modulePath == "" {
		return out, nil
	}
	err := filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		base := d.Name()
		if base == "vendor" || base == "node_modules" || base == ".git" || base == ".forst" {
			return filepath.SkipDir
		}
		if _, skip := skipDirs[path]; skip {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		hasGo, hasFt := false, false
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			n := e.Name()
			if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, ".gen.go") {
				hasGo = true
			}
			if strings.HasSuffix(n, ".ft") {
				hasFt = true
			}
		}
		if !hasGo || hasFt {
			return nil
		}
		rel, err := filepath.Rel(moduleRoot, path)
		if err != nil || rel == "." {
			return nil
		}
		imp := modulePath + "/" + filepath.ToSlash(rel)
		out = append(out, gowork.PackageReplace{ImportPath: imp, Dir: path})
		return nil
	})
	return out, err
}
