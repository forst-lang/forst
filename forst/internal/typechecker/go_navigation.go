package typechecker

import (
	"go/types"

	"forst/internal/ast"
	"forst/internal/goload"
)

// GoImportPathDefinitionLocation returns a file-level location for a Go import path string.
func (tc *TypeChecker) GoImportPathDefinitionLocation(importPath string) (goload.SourceLocation, bool) {
	if tc == nil || importPath == "" {
		return goload.SourceLocation{}, false
	}
	_, goPkg, err := goload.LoadDocPackage(tc.goPackagesLoadDir(), importPath)
	if err != nil || goPkg == nil {
		return goload.SourceLocation{}, false
	}
	return goload.PackageFileLocation(goPkg)
}

// GoQualifiedExportDefinitionLocation returns the definition span for pkgLocal.symbol (e.g. exec.Command).
func (tc *TypeChecker) GoQualifiedExportDefinitionLocation(pkgLocal, symbol string) (goload.SourceLocation, bool) {
	if tc == nil || pkgLocal == "" || symbol == "" || tc.IsNodeImportLocal(pkgLocal) {
		return goload.SourceLocation{}, false
	}
	importPath, ok := tc.ImportPathForLocal(pkgLocal)
	if !ok || importPath == "" || !tc.IsImportedLocalName(pkgLocal) {
		return goload.SourceLocation{}, false
	}
	gp := tc.goPackageForImportLocal(pkgLocal)
	if gp == nil {
		return goload.SourceLocation{}, false
	}
	obj := gp.Scope().Lookup(symbol)
	if obj == nil {
		return goload.SourceLocation{}, false
	}
	return tc.goObjectDefinitionLocation(importPath, obj)
}

// GoImportLocalDefinitionLocation returns a file-level location for a Go import local name.
func (tc *TypeChecker) GoImportLocalDefinitionLocation(pkgLocal string) (goload.SourceLocation, bool) {
	if tc == nil || pkgLocal == "" || tc.IsNodeImportLocal(pkgLocal) {
		return goload.SourceLocation{}, false
	}
	importPath, ok := tc.ImportPathForLocal(pkgLocal)
	if !ok || importPath == "" || !tc.IsImportedLocalName(pkgLocal) {
		return goload.SourceLocation{}, false
	}
	return tc.GoImportPathDefinitionLocation(importPath)
}

// GoSamePackageFuncDefinitionLocation returns the definition span for an exported same-package Go func.
func (tc *TypeChecker) GoSamePackageFuncDefinitionLocation(symbol string) (goload.SourceLocation, bool) {
	if tc == nil || tc.samePackageGo == nil || tc.samePackageGoImportPath == "" || symbol == "" {
		return goload.SourceLocation{}, false
	}
	if !goIdentifierExported(symbol) {
		return goload.SourceLocation{}, false
	}
	obj := tc.samePackageGo.Scope().Lookup(symbol)
	if obj == nil {
		return goload.SourceLocation{}, false
	}
	if _, ok := obj.(*types.Func); !ok {
		return goload.SourceLocation{}, false
	}
	return tc.goObjectDefinitionLocation(tc.samePackageGoImportPath, obj)
}

// GoDotImportFuncDefinitionLocation returns the definition span for a unique dot-imported func.
func (tc *TypeChecker) GoDotImportFuncDefinitionLocation(symbol string) (goload.SourceLocation, bool) {
	if tc == nil || symbol == "" {
		return goload.SourceLocation{}, false
	}
	gp := tc.dotImportPackageForUniqueFunc(symbol)
	if gp == nil {
		return goload.SourceLocation{}, false
	}
	obj := gp.Scope().Lookup(symbol)
	if obj == nil {
		return goload.SourceLocation{}, false
	}
	return tc.goObjectDefinitionLocation(gp.Path(), obj)
}

// GoReceiverMethodDefinitionLocation returns the definition span for recvLocal.methodName.
func (tc *TypeChecker) GoReceiverMethodDefinitionLocation(recvLocal, methodName string) (goload.SourceLocation, bool) {
	if tc == nil || recvLocal == "" || methodName == "" {
		return goload.SourceLocation{}, false
	}
	if tc.IsImportedLocalName(recvLocal) || tc.IsNodeImportLocal(recvLocal) {
		return goload.SourceLocation{}, false
	}
	gt := tc.GoTypeForVariable(ast.Identifier(recvLocal))
	if gt == nil {
		return goload.SourceLocation{}, false
	}
	importPath := tc.importPathForGoType(gt)
	if importPath == "" {
		importPath = tc.samePackageGoImportPath
	}
	if importPath == "" {
		return goload.SourceLocation{}, false
	}
	docPkg, goPkg, err := goload.LoadDocPackage(tc.goPackagesLoadDir(), importPath)
	if err != nil || goPkg == nil || goPkg.Fset == nil || goPkg.Types == nil {
		return goload.SourceLocation{}, false
	}
	recvType := receiverTypeInPackage(gt, goPkg.Types)
	if typeName := namedGoTypeBaseName(recvType); typeName != "" {
		if loc, ok := goload.MethodDeclLocation(docPkg, goPkg.Fset, typeName, methodName); ok {
			return loc, true
		}
	}
	obj, _, _ := types.LookupFieldOrMethod(recvType, true, goPkg.Types, methodName)
	if obj == nil {
		return goload.SourceLocation{}, false
	}
	if loc, ok := goload.SourceLocationForObject(goPkg.Fset, obj); ok {
		return loc, true
	}
	return goload.SourceLocation{}, false
}

func receiverTypeInPackage(gt types.Type, pkg *types.Package) types.Type {
	if gt == nil || pkg == nil {
		return gt
	}
	name := namedGoTypeBaseName(gt)
	if name == "" {
		return gt
	}
	obj := pkg.Scope().Lookup(name)
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return gt
	}
	t := tn.Type()
	if _, ok := gt.(*types.Pointer); ok {
		return types.NewPointer(t)
	}
	return t
}

func namedGoTypeBaseName(gt types.Type) string {
	if ptr, ok := gt.(*types.Pointer); ok {
		gt = ptr.Elem()
	}
	if named, ok := gt.(*types.Named); ok && named.Obj() != nil {
		return named.Obj().Name()
	}
	return ""
}

func (tc *TypeChecker) goObjectDefinitionLocation(importPath string, obj types.Object) (goload.SourceLocation, bool) {
	if tc == nil || importPath == "" || obj == nil {
		return goload.SourceLocation{}, false
	}
	_, goPkg, err := goload.LoadDocPackage(tc.goPackagesLoadDir(), importPath)
	if err != nil || goPkg == nil || goPkg.Fset == nil || goPkg.Types == nil {
		return goload.SourceLocation{}, false
	}
	lookup := obj
	if name := obj.Name(); name != "" {
		if pkgObj := goPkg.Types.Scope().Lookup(name); pkgObj != nil {
			lookup = pkgObj
		}
	}
	if loc, ok := goload.SourceLocationForObject(goPkg.Fset, lookup); ok {
		return loc, true
	}
	return goload.PackageFileLocation(goPkg)
}
