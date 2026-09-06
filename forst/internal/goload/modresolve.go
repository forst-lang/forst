package goload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const listModulesTimeout = 60 * time.Second

type moduleListEntry struct {
	Path    string
	Version string
	Dir     string
	Replace *moduleListEntry
}

// ResolveImportDir finds the filesystem directory for importPath using the module graph.
// Works for Forst-only packages (no .go files) that packages.Load cannot locate.
func ResolveImportDir(moduleRoot, importPath string) (PackageLoc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listModulesTimeout)
	defer cancel()
	return ResolveImportDirContext(ctx, moduleRoot, importPath)
}

// ResolveImportDirContext is ResolveImportDir with an explicit context (deadline/cancel).
func ResolveImportDirContext(ctx context.Context, moduleRoot, importPath string) (PackageLoc, error) {
	if moduleRoot == "" || importPath == "" {
		return PackageLoc{}, fmt.Errorf("resolve import dir: empty args")
	}
	moduleRoot = FindModuleRoot(moduleRoot)
	mods, err := listModules(ctx, moduleRoot)
	if err != nil {
		return PackageLoc{}, err
	}
	bestLen := -1
	var best moduleListEntry
	for _, m := range mods {
		path := m.Path
		dir := m.Dir
		ver := m.Version
		if m.Replace != nil {
			if m.Replace.Dir != "" {
				dir = m.Replace.Dir
			}
			if m.Replace.Path != "" && !strings.Contains(m.Replace.Path, "/") && !strings.HasPrefix(m.Replace.Path, ".") {
				// replaced by another module path — keep require path for matching
			}
		}
		if path == "" || dir == "" {
			continue
		}
		if importPath == path || strings.HasPrefix(importPath, path+"/") {
			if len(path) > bestLen {
				bestLen = len(path)
				best = moduleListEntry{Path: path, Version: ver, Dir: dir}
			}
		}
	}
	if bestLen < 0 {
		return PackageLoc{}, fmt.Errorf("resolve import dir: no module for %q", importPath)
	}
	rel := strings.TrimPrefix(importPath, best.Path)
	rel = strings.TrimPrefix(rel, "/")
	pkgDir := best.Dir
	if rel != "" {
		for _, seg := range strings.Split(rel, "/") {
			if seg == ".." {
				return PackageLoc{}, fmt.Errorf("resolve import dir: invalid path %q", importPath)
			}
		}
		pkgDir = filepath.Join(best.Dir, filepath.FromSlash(rel))
	}
	moduleDir := best.Dir
	if moduleDir != "" {
		moduleDir = filepath.Clean(moduleDir)
	}
	return PackageLoc{
		ImportPath:    importPath,
		Dir:           filepath.Clean(pkgDir),
		ModulePath:    best.Path,
		ModuleVersion: best.Version,
		ModuleDir:     moduleDir,
	}, nil
}

func listModules(ctx context.Context, moduleRoot string) ([]moduleListEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, listModulesTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	cmd.Dir = moduleRoot
	cmd.Env = loadPackagesEnv(moduleRoot)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list -m: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	dec := json.NewDecoder(&stdout)
	var out []moduleListEntry
	for dec.More() {
		var m moduleListEntry
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}
