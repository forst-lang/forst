package goload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type moduleListEntry struct {
	Path    string
	Version string
	Dir     string
	Replace *moduleListEntry
}

// ResolveImportDir finds the filesystem directory for importPath using the module graph.
// Works for Forst-only packages (no .go files) that packages.Load cannot locate.
func ResolveImportDir(moduleRoot, importPath string) (PackageLoc, error) {
	if moduleRoot == "" || importPath == "" {
		return PackageLoc{}, fmt.Errorf("resolve import dir: empty args")
	}
	moduleRoot = FindModuleRoot(moduleRoot)
	mods, err := listModules(moduleRoot)
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
		pkgDir = filepath.Join(best.Dir, filepath.FromSlash(rel))
	}
	return PackageLoc{
		ImportPath:    importPath,
		Dir:           filepath.Clean(pkgDir),
		ModulePath:    best.Path,
		ModuleVersion: best.Version,
		ModuleDir:     filepath.Clean(best.Dir),
	}, nil
}

func listModules(moduleRoot string) ([]moduleListEntry, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
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
