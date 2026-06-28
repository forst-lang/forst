// Package goload loads Go packages (stdlib and module deps) for Forst↔Go type checking via go/packages.
package goload

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

type loadByPkgPathCacheEntry struct {
	out map[string]*packages.Package
	err error
}

var loadByPkgPathCache sync.Map

// ClearLoadCacheForTest drops cached LoadByPkgPath results (for tests that mutate module dirs).
func ClearLoadCacheForTest() {
	loadByPkgPathCache = sync.Map{}
}

func loadByPkgPathCacheKey(dir string, importPaths []string) string {
	sorted := append([]string(nil), importPaths...)
	sort.Strings(sorted)
	return filepath.Clean(dir) + "\x00" + strings.Join(sorted, "\x00")
}

func storeLoadCache(dir string, importPaths []string, out map[string]*packages.Package, err error) {
	loadByPkgPathCache.Store(loadByPkgPathCacheKey(dir, importPaths), loadByPkgPathCacheEntry{out: out, err: err})
	if err != nil || out == nil {
		return
	}
	for path, p := range out {
		if path == "" {
			continue
		}
		singleKey := loadByPkgPathCacheKey(dir, []string{path})
		singleOut := map[string]*packages.Package{path: p}
		loadByPkgPathCache.Store(singleKey, loadByPkgPathCacheEntry{out: singleOut, err: nil})
	}
}

// LoadByPkgPath loads packages by import path (e.g. "fmt", "strconv") and returns import path -> *packages.Package.
// Results are cached per (module root Dir, import path set) for the process lifetime; single-path loads reuse batch results.
func LoadByPkgPath(dir string, importPaths []string) (map[string]*packages.Package, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
	dir = FindModuleRoot(dir)
	key := loadByPkgPathCacheKey(dir, importPaths)
	if v, ok := loadByPkgPathCache.Load(key); ok {
		e := v.(loadByPkgPathCacheEntry)
		return e.out, e.err
	}
	out, err := loadByPkgPathUncached(dir, importPaths)
	storeLoadCache(dir, importPaths, out, err)
	return out, err
}

// packageHasGoSources reports whether p has Go source files (real package vs directory placeholder).
func packageHasGoSources(p *packages.Package) bool {
	if p == nil {
		return false
	}
	return len(p.GoFiles)+len(p.CompiledGoFiles)+len(p.OtherFiles) > 0
}

// PackageLoadOK reports whether p is a successfully resolved Go package for importPath.
// go/packages may return placeholder packages when an import cannot be resolved to source
// (e.g. a Forst-only sibling directory with no .go files); those have an empty Types.Name().
// Use Types.Path() as the canonical import path (stdlib packages may leave PkgPath unset).
// moduleRoot scopes extra source-file checks to packages under the local module path.
func PackageLoadOK(p *packages.Package, importPath string) bool {
	return packageLoadOKAt(p, importPath, "")
}

func packageLoadOKAt(p *packages.Package, importPath, moduleRoot string) bool {
	if p == nil || p.Types == nil || importPath == "" {
		return false
	}
	if p.Types.Name() == "" {
		return false
	}
	if p.Types.Path() != importPath {
		return false
	}
	modPath := ModulePath(moduleRoot)
	if modPath != "" && strings.HasPrefix(importPath, modPath+"/") {
		return packageHasGoSources(p)
	}
	return true
}

func loadPackagesEnv(dir string) []string {
	env := os.Environ()
	if ModulePath(dir) != "" {
		env = append(env, "GOWORK=off")
	}
	return env
}

func loadByPkgPathUncached(dir string, importPaths []string) (map[string]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports,
		Dir:  dir,
		Env:  loadPackagesEnv(dir),
	}
	pkgs, err := packages.Load(cfg, importPaths...)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*packages.Package, len(importPaths))
	for _, p := range pkgs {
		path := ""
		if p.Types != nil {
			path = p.Types.Path()
		}
		if !packageLoadOKAt(p, path, dir) {
			continue
		}
		out[path] = p
	}
	if len(out) == 0 {
		var b strings.Builder
		for _, p := range pkgs {
			for _, e := range p.Errors {
				b.WriteString(e.Msg)
				b.WriteByte('\n')
			}
		}
		if b.Len() > 0 {
			return nil, fmt.Errorf("packages.Load: no typed packages for %v: %s", importPaths, b.String())
		}
		return nil, fmt.Errorf("packages.Load: no typed packages for %v", importPaths)
	}
	return out, nil
}

// ImportPathFromForst returns the Go import path string from a Forst import literal (quoted or bare).
func ImportPathFromForst(path string) string {
	s := strings.TrimSpace(path)
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	return s
}
