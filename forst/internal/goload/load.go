// Package goload loads Go packages (stdlib and module deps) for Forst↔Go type checking via go/packages.
package goload

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

// LoadByPkgPath loads packages by import path (e.g. "fmt", "strconv") and returns PkgPath -> *types.Package.
func LoadByPkgPath(dir string, importPaths []string) (map[string]*packages.Package, error) {
	if len(importPaths) == 0 {
		return nil, nil
	}
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
