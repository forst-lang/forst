package goload

import (
	"fmt"
	"strings"
	"sync"

	goast "go/ast"
	"go/doc"
	"go/doc/comment"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type docPackageCacheEntry struct {
	docPkg *doc.Package
	goPkg  *packages.Package
	err    error
}

var docPackageCache sync.Map

// ClearDocCacheForTest drops cached LoadDocPackage results.
func ClearDocCacheForTest() {
	clearSyncMap(&docPackageCache)
}

func docPackageCacheKey(moduleRoot, importPath string) string {
	return FindModuleRoot(moduleRoot) + "\x00" + importPath
}

// LoadDocPackage loads go/doc metadata for importPath (lazy, cached separately from LoadByPkgPath).
func LoadDocPackage(moduleRoot, importPath string) (*doc.Package, *packages.Package, error) {
	if importPath == "" {
		return nil, nil, fmt.Errorf("empty import path")
	}
	moduleRoot = FindModuleRoot(moduleRoot)
	key := docPackageCacheKey(moduleRoot, importPath)
	if v, ok := docPackageCache.Load(key); ok {
		e := v.(docPackageCacheEntry)
		return e.docPkg, e.goPkg, e.err
	}
	docPkg, goPkg, err := loadDocPackageUncached(moduleRoot, importPath)
	docPackageCache.Store(key, docPackageCacheEntry{docPkg: docPkg, goPkg: goPkg, err: err})
	return docPkg, goPkg, err
}

func loadDocPackageUncached(moduleRoot, importPath string) (*doc.Package, *packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes,
		Dir:  moduleRoot,
		Env:  loadPackagesEnv(moduleRoot),
	}
	pkgs, err := packagesLoadFn(cfg, importPath)
	if err != nil {
		return nil, nil, err
	}
	var goPkg *packages.Package
	for _, p := range pkgs {
		if p == nil || p.Types == nil {
			continue
		}
		path := p.Types.Path()
		if path == importPath && packageLoadOKAt(p, path, moduleRoot) {
			goPkg = p
			break
		}
	}
	if goPkg == nil {
		return nil, nil, fmt.Errorf("doc load: package %q not found", importPath)
	}
	paths := goPkg.CompiledGoFiles
	if len(paths) == 0 {
		paths = goPkg.GoFiles
	}
	files := make([]*goast.File, 0, len(goPkg.Syntax))
	for i, f := range goPkg.Syntax {
		if f == nil {
			continue
		}
		if i < len(paths) && strings.HasSuffix(paths[i], "_test.go") {
			continue
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, goPkg, fmt.Errorf("doc load: no syntax for %q", importPath)
	}
	docPkg, err := doc.NewFromFiles(goPkg.Fset, files, goPkg.Name, doc.AllDecls)
	if err != nil {
		return nil, goPkg, err
	}
	return docPkg, goPkg, nil
}

// FormatDocMarkdown renders a godoc comment string as markdown.
func FormatDocMarkdown(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var cdocParser comment.Parser
	var pr comment.Printer
	return strings.TrimSpace(string(pr.Markdown(cdocParser.Parse(raw))))
}

// DocForFunc returns documentation for an exported package-level function.
func DocForFunc(docPkg *doc.Package, name string) string {
	if docPkg == nil || name == "" {
		return ""
	}
	for _, f := range docPkg.Funcs {
		if f.Name == name {
			return FormatDocMarkdown(f.Doc)
		}
	}
	return ""
}

// DocForType returns documentation for an exported named type.
func DocForType(docPkg *doc.Package, typeName string) string {
	if docPkg == nil || typeName == "" {
		return ""
	}
	for _, t := range docPkg.Types {
		if t.Name == typeName {
			return FormatDocMarkdown(t.Doc)
		}
	}
	return ""
}

// DocForMethod returns documentation for a method on a named type.
func DocForMethod(docPkg *doc.Package, recvTypeName, methodName string) string {
	if docPkg == nil || recvTypeName == "" || methodName == "" {
		return ""
	}
	for _, t := range docPkg.Types {
		if t.Name != recvTypeName {
			continue
		}
		for _, m := range t.Methods {
			if m.Name == methodName {
				if d := FormatDocMarkdown(m.Doc); d != "" {
					return d
				}
				return FormatDocMarkdown(t.Doc)
			}
		}
	}
	return ""
}

// DocForObject returns godoc for a types.Object when doc metadata is available.
func DocForObject(docPkg *doc.Package, obj types.Object) string {
	if docPkg == nil || obj == nil {
		return ""
	}
	switch obj.(type) {
	case *types.Func:
		fn := obj.(*types.Func)
		if recv := fn.Signature().Recv(); recv != nil {
			recvType := recv.Type()
			typeName := types.TypeString(recvType, func(*types.Package) string { return "" })
			typeName = strings.TrimPrefix(typeName, "*")
			if idx := strings.LastIndex(typeName, "."); idx >= 0 {
				typeName = typeName[idx+1:]
			}
			return DocForMethod(docPkg, typeName, fn.Name())
		}
		return DocForFunc(docPkg, fn.Name())
	case *types.Const, *types.Var:
		// Vars/consts at package level may appear in docPkg.Vars
		name := obj.Name()
		for _, v := range docPkg.Vars {
			for _, n := range v.Names {
				if n == name {
					return FormatDocMarkdown(v.Doc)
				}
			}
		}
	case *types.TypeName:
		return DocForType(docPkg, obj.Name())
	}
	return ""
}

// PkgGoDevURL returns a pkg.go.dev link for an import path and optional symbol fragment.
func PkgGoDevURL(importPath, symbol string) string {
	if importPath == "" {
		return ""
	}
	url := "https://pkg.go.dev/" + importPath
	if symbol != "" {
		url += "#" + symbol
	}
	return url
}

// IsStdlibImportPath reports whether path looks like a standard library import (no dot).
func IsStdlibImportPath(path string) bool {
	return path != "" && !strings.Contains(path, ".")
}
