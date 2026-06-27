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

// LoadByPkgPath loads packages by import path (e.g. "fmt", "strconv") and returns PkgPath -> *types.Package.
// Results are cached per (Dir, import path set) for the process lifetime.
func LoadByPkgPath(dir string, importPaths []string) (map[string]*packages.Package, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
	dir = filepath.Clean(dir)
	key := loadByPkgPathCacheKey(dir, importPaths)
	if v, ok := loadByPkgPathCache.Load(key); ok {
		e := v.(loadByPkgPathCacheEntry)
		return e.out, e.err
	}
	out, err := loadByPkgPathUncached(dir, importPaths)
	loadByPkgPathCache.Store(key, loadByPkgPathCacheEntry{out: out, err: err})
	return out, err
}

func loadByPkgPathUncached(dir string, importPaths []string) (map[string]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedDeps | packages.NeedImports,
		Dir:  dir,
		Env:  os.Environ(),
	}
	pkgs, err := packages.Load(cfg, importPaths...)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*packages.Package, len(pkgs))
	for _, p := range pkgs {
		if p.Types == nil {
			continue
		}
		key := p.PkgPath
		if key == "" {
			key = p.ID
		}
		if key != "" {
			out[key] = p
		}
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
