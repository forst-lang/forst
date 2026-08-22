package importlocal

import (
	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/typechecker/gointerop"
)

// Binding holds a resolved import local name and metadata.
type Binding struct {
	Local      string
	ImportPath string
	ModuleID   string
	GoPath     string
	Skip       bool
}

// BindingOpts configures BindingFromAST.
type BindingOpts struct {
	Kind      Kind
	ModuleID  string
	GoPkgName string
}

// BindingFromAST resolves the import local from an AST import node.
func BindingFromAST(imp ast.ImportNode, opts BindingOpts) Binding {
	b := Binding{
		ImportPath: imp.Path,
		ModuleID:   opts.ModuleID,
	}

	if imp.Alias != nil {
		switch imp.Alias.ID {
		case ".":
			b.GoPath, _ = goPathAndFallbackLocal(imp)
			b.Skip = true
			return b
		case "_":
			b.GoPath, _ = goPathAndFallbackLocal(imp)
			b.Skip = true
			return b
		}
	}

	switch opts.Kind {
	case KindGo:
		goPath, fallbackLocal := goPathAndFallbackLocal(imp)
		b.GoPath = goPath
		if goPath == "" {
			b.Skip = true
			return b
		}
		if imp.Alias != nil {
			b.Local = string(imp.Alias.ID)
		} else if opts.GoPkgName != "" {
			b.Local = opts.GoPkgName
		} else {
			b.Local = fallbackLocal
		}
		if b.ModuleID == "" {
			b.ModuleID = goPath
		}
	case KindNode:
		if imp.Alias != nil {
			b.Local = string(imp.Alias.ID)
		} else if opts.ModuleID != "" {
			b.Local = DefaultLocalFromModuleID(opts.ModuleID)
		} else {
			b.Local = DefaultLocalFromModuleID(imp.Path)
		}
	default:
		if imp.Alias != nil {
			b.Local = string(imp.Alias.ID)
		}
	}

	if b.Local == "" {
		b.Skip = true
	}
	return b
}

func goPathAndFallbackLocal(imp ast.ImportNode) (goPath, local string) {
	return gointerop.FallbackImportLocal(imp)
}

// GoModuleID returns the normalized Go import path for imp.Path.
func GoModuleID(importPath string) string {
	return goload.ImportPathFromForst(importPath)
}
