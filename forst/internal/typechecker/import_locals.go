package typechecker

import (
	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/importlocal"

	"golang.org/x/tools/go/packages"
)

type importLocalContext struct {
	goImports   []ast.ImportNode
	nodeImports []ast.ImportNode
	nodeByLocal map[string]nodeImportBinding
}

func (tc *TypeChecker) importLocalContext() importLocalContext {
	return importLocalContext{
		goImports:   tc.imports,
		nodeImports: tc.nodeImports,
		nodeByLocal: tc.nodeImportsByLocal,
	}
}

func (ctx importLocalContext) takenSet() importlocal.TakenSet {
	taken := make(importlocal.TakenSet)
	for _, imp := range ctx.goImports {
		b := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
			Kind:     importlocal.KindGo,
			ModuleID: importlocal.GoModuleID(imp.Path),
		})
		if !b.Skip {
			taken.Add(b.Local)
		}
	}
	for _, imp := range ctx.nodeImports {
		b := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
			Kind:     importlocal.KindBridge,
			ModuleID: imp.Path,
		})
		if !b.Skip {
			taken.Add(b.Local)
		}
	}
	for local := range ctx.nodeByLocal {
		taken.Add(local)
	}
	return taken
}

func (ctx importLocalContext) checkReserved(binding importlocal.Binding, kind importlocal.Kind, span ast.SourceSpan) error {
	if binding.Skip || binding.Local == "" {
		return nil
	}
	if err := importlocal.Validate(binding.Local, kind); err != nil {
		moduleID := binding.ModuleID
		if moduleID == "" {
			moduleID = binding.ImportPath
		}
		msg := importlocal.ReservedLocalDiagnostic(binding.Local, binding.ImportPath, moduleID, ctx.takenSet(), kind, err)
		code := "js-import-reserved-local"
		if kind == importlocal.KindGo {
			code = "go-import-reserved-local"
		}
		return reportBodyf(span, code, "%s", msg)
	}
	return nil
}

func (ctx importLocalContext) checkCrossKindConflict(local string, kind importlocal.Kind, span ast.SourceSpan) error {
	if local == "" {
		return nil
	}
	switch kind {
	case importlocal.KindGo:
		for _, imp := range ctx.nodeImports {
			b := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
				Kind:     importlocal.KindBridge,
				ModuleID: imp.Path,
			})
			if !b.Skip && b.Local == local {
				return reportBodyf(span, "go-import", "Go import local name %q conflicts with JS import", local)
			}
		}
		if _, ok := ctx.nodeByLocal[local]; ok {
			return reportBodyf(span, "go-import", "Go import local name %q conflicts with JS import", local)
		}
	case importlocal.KindBridge:
		for _, imp := range ctx.goImports {
			b := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
				Kind:     importlocal.KindGo,
				ModuleID: importlocal.GoModuleID(imp.Path),
			})
			if !b.Skip && b.Local == local {
				return reportBodyf(span, "js-import", "JS import local name %q conflicts with Go import", local)
			}
		}
	}
	return nil
}

func (ctx importLocalContext) checkDuplicate(local, pathKey string, kind importlocal.Kind, seen map[string]string, span ast.SourceSpan) error {
	if local == "" {
		return nil
	}
	if prev, dup := seen[local]; dup && prev != pathKey {
		switch kind {
		case importlocal.KindGo:
			return reportBodyf(span, "go-import", "duplicate Go import local name %q", local)
		default:
			return reportBodyf(span, "js-import", "duplicate JS import local name %q", local)
		}
	}
	seen[local] = pathKey
	return nil
}

func (tc *TypeChecker) validateGoImportAtCollect(imp ast.ImportNode) error {
	if imp.Alias == nil || imp.Alias.ID == "." || imp.Alias.ID == "_" {
		return nil
	}
	ctx := tc.importLocalContext()
	b := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
		Kind:     importlocal.KindGo,
		ModuleID: importlocal.GoModuleID(imp.Path),
	})
	span := importNodeSpan(imp)
	if err := ctx.checkReserved(b, importlocal.KindGo, span); err != nil {
		return err
	}
	if err := ctx.checkCrossKindConflict(b.Local, importlocal.KindGo, span); err != nil {
		return err
	}
	for _, existing := range tc.imports {
		eb := importlocal.BindingFromAST(existing, importlocal.BindingOpts{
			Kind:     importlocal.KindGo,
			ModuleID: importlocal.GoModuleID(existing.Path),
		})
		if eb.Skip || eb.Local != b.Local {
			continue
		}
		return reportBodyf(span, "go-import", "duplicate Go import local name %q", b.Local)
	}
	return nil
}

func (tc *TypeChecker) validateGoImportLocalsAfterLoad(loaded map[string]*packages.Package) error {
	ctx := tc.importLocalContext()
	seen := make(map[string]string)
	for _, imp := range tc.imports {
		goPath := importlocal.GoModuleID(imp.Path)
		b := goBindingFromLoaded(imp, loaded)
		if b.Skip {
			continue
		}
		if err := ctx.checkReserved(b, importlocal.KindGo, importNodeSpan(imp)); err != nil {
			return err
		}
		if err := ctx.checkCrossKindConflict(b.Local, importlocal.KindGo, importNodeSpan(imp)); err != nil {
			return err
		}
		if err := ctx.checkDuplicate(b.Local, goPath, importlocal.KindGo, seen, importNodeSpan(imp)); err != nil {
			return err
		}
	}
	return nil
}

func goBindingFromLoaded(imp ast.ImportNode, loaded map[string]*packages.Package) importlocal.Binding {
	opts := importlocal.BindingOpts{
		Kind:     importlocal.KindGo,
		ModuleID: importlocal.GoModuleID(imp.Path),
	}
	ip := opts.ModuleID
	if ip != "" && loaded != nil {
		if pkgp, ok := loaded[ip]; ok && goload.PackageLoadOK(pkgp, ip) && pkgp.Types != nil {
			opts.GoPkgName = pkgp.Types.Name()
		}
	}
	return importlocal.BindingFromAST(imp, opts)
}

func (tc *TypeChecker) validateNodeImportLocal(binding importlocal.Binding, seen map[string]string, imp ast.ImportNode) error {
	ctx := tc.importLocalContext()
	span := importNodeSpan(imp)
	if err := ctx.checkReserved(binding, importlocal.KindBridge, span); err != nil {
		return err
	}
	if err := ctx.checkCrossKindConflict(binding.Local, importlocal.KindBridge, span); err != nil {
		return err
	}
	return ctx.checkDuplicate(binding.Local, binding.ModuleID, importlocal.KindBridge, seen, span)
}

func (tc *TypeChecker) registerImportLocalsFromAST() {
	for _, imp := range tc.imports {
		b := importlocal.BindingFromAST(imp, importlocal.BindingOpts{
			Kind:     importlocal.KindGo,
			ModuleID: importlocal.GoModuleID(imp.Path),
		})
		if b.Skip || b.Local == "" {
			continue
		}
		if prev, dup := tc.importPathByLocal[b.Local]; dup && prev != b.GoPath {
			continue
		}
		tc.importPathByLocal[b.Local] = b.GoPath
	}
}
