package typechecker

import (
	"fmt"
	"go/doc/comment"
	"go/types"
	"strings"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/hoverdoc"
)

// IsImportedLocalName reports whether id is a Go import's local identifier in this file.
func (tc *TypeChecker) IsImportedLocalName(id string) bool {
	if tc.importPathByLocal == nil {
		return false
	}
	_, ok := tc.importPathByLocal[id]
	return ok
}

// ImportPathForLocal returns the Go import path for an import's local name (e.g. users → interopdemo/users).
// Resolution uses the import map from go/packages when available, then falls back to Forst import lines in the AST.
func (tc *TypeChecker) ImportPathForLocal(local string) (string, bool) {
	if local == "" {
		return "", false
	}
	if tc.importPathByLocal != nil {
		if path, ok := tc.importPathByLocal[local]; ok && path != "" {
			return path, true
		}
	}
	for _, imp := range tc.imports {
		path, impLocal := fallbackImportLocal(imp)
		if impLocal == local && path != "" {
			return path, true
		}
	}
	return "", false
}

// ImportLocalForPath returns the import local name for a Go import path (inverse of ImportPathForLocal).
func (tc *TypeChecker) ImportLocalForPath(path string) (string, bool) {
	if tc.importPathByLocal == nil || path == "" {
		return "", false
	}
	for local, p := range tc.importPathByLocal {
		if p == path {
			return local, true
		}
	}
	return "", false
}

// loadedGoPackageByImportPath returns a loaded *types.Package for the given import path, including
// dot-imported packages and lazy-loaded regular imports (same as goPackageForImportLocal).
func (tc *TypeChecker) loadedGoPackageByImportPath(path string) *types.Package {
	if path == "" {
		return nil
	}
	if tc.importPathByLocal != nil {
		for local, p := range tc.importPathByLocal {
			if p == path {
				if gp := tc.goPackageForImportLocal(local); gp != nil {
					return gp
				}
			}
		}
	}
	for _, dp := range tc.dotImportPkgs {
		if dp == nil {
			continue
		}
		if dp.Path() == path {
			return dp
		}
	}
	return nil
}

func (tc *TypeChecker) dotImportPackageForUniqueFunc(symbol string) *types.Package {
	if len(tc.dotImportPkgs) == 0 || !goIdentifierExported(symbol) {
		return nil
	}
	var matched []*types.Package
	for _, pkg := range tc.dotImportPkgs {
		if pkg == nil {
			continue
		}
		obj := pkg.Scope().Lookup(symbol)
		if obj == nil {
			continue
		}
		if _, ok := obj.(*types.Func); !ok {
			continue
		}
		matched = append(matched, pkg)
	}
	if len(matched) != 1 {
		return nil
	}
	return matched[0]
}

func (tc *TypeChecker) goDocParagraphForObject(importPath string, obj types.Object) string {
	if tc == nil || obj == nil || importPath == "" {
		return ""
	}
	docPkg, _, err := goload.LoadDocPackage(tc.GoWorkspaceDir, importPath)
	if err != nil || docPkg == nil {
		return ""
	}
	return goload.DocForObject(docPkg, obj)
}

func (tc *TypeChecker) goHoverHeader(importPath, pkgQual, symbol string) string {
	if goload.IsStdlibImportPath(importPath) {
		if symbol != "" {
			return fmt.Sprintf("**Go** `%s.%s` — [package %s](https://pkg.go.dev/%s)", pkgQual, symbol, importPath, importPath)
		}
		return fmt.Sprintf("**Go package** `%s` — [pkg.go.dev/%s](https://pkg.go.dev/%s)", pkgQual, importPath, importPath)
	}
	url := goload.PkgGoDevURL(importPath, symbol)
	if symbol != "" {
		return fmt.Sprintf("**Go** `%s.%s` — [%s](%s)", pkgQual, symbol, importPath, url)
	}
	return fmt.Sprintf("**Go package** `%s` — [%s](%s)", pkgQual, importPath, url)
}

func (tc *TypeChecker) goHoverMarkdownBodyForResolvedPackage(gp *types.Package, importPath, pkgQual, symbol string) (string, bool) {
	obj := gp.Scope().Lookup(symbol)
	if obj == nil {
		return fmt.Sprintf("**Go** `%s.%s` — not found in `%s`", pkgQual, symbol, gp.Path()), true
	}

	qf := types.RelativeTo(gp)
	head := types.ObjectString(obj, qf)
	var b strings.Builder
	if importPath == "" {
		importPath = gp.Path()
	}
	b.WriteString(tc.goHoverHeader(importPath, pkgQual, symbol))
	b.WriteString("\n\n")
	if docText := tc.goDocParagraphForObject(importPath, obj); docText != "" {
		b.WriteString(docText)
		b.WriteString("\n\n")
	}
	b.WriteString(hoverdoc.GoBlock(head))

	qual := pkgQual + "." + symbol
	if bfn, ok := BuiltinFunctions[qual]; ok {
		b.WriteString("\n\n**Forst (builtin table)** ")
		if bfn.HoverSignature != "" {
			b.WriteString(bfn.HoverSignature + " → ")
		} else if len(bfn.ParamTypes) > 0 {
			parts := make([]string, len(bfn.ParamTypes))
			for i, p := range bfn.ParamTypes {
				parts[i] = tc.FormatTypeNodeDisplay(p)
			}
			b.WriteString("params `(" + strings.Join(parts, ", ") + ")` → ")
		}
		b.WriteString("return `" + tc.FormatTypeNodeDisplay(bfn.ReturnType) + "`")
	} else if fn, ok := obj.(*types.Func); ok {
		if sig, ok2 := fn.Type().(*types.Signature); ok2 {
			res := sig.Results()
			if res.Len() > 0 {
				qf := types.RelativeTo(gp)
				parts := make([]string, res.Len())
				for i := 0; i < res.Len(); i++ {
					parts[i] = types.TypeString(res.At(i).Type(), qf)
				}
				b.WriteString("\n\n**Go return types**\n\n")
				b.WriteString(hoverdoc.GoBlock(strings.Join(parts, ", ")))
			}
			if mapped := goSignatureReturnsToForst(sig); len(mapped) > 0 {
				b.WriteString("\n\n**Forst-mapped returns**\n\n")
				b.WriteString(hoverdoc.ForstBlock(mappedForstReturnLines(tc, mapped)...))
			}
		}
	}

	return b.String(), true
}

// GoHoverMarkdownDotImportedSymbol returns hover for an unqualified call that resolves to a unique
// dot-imported package function (e.g. Contains after import . "strings").
func (tc *TypeChecker) GoHoverMarkdownDotImportedSymbol(symbol string) (string, bool) {
	gp := tc.dotImportPackageForUniqueFunc(symbol)
	if gp == nil {
		return "", false
	}
	return tc.goHoverMarkdownBodyForResolvedPackage(gp, gp.Path(), gp.Name(), symbol)
}

// GoHoverMarkdownSamePackageFunc returns hover for an unqualified exported func in the same directory.
func (tc *TypeChecker) GoHoverMarkdownSamePackageFunc(symbol string) (string, bool) {
	if tc == nil || tc.samePackageGo == nil || tc.samePackageGoImportPath == "" || symbol == "" {
		return "", false
	}
	if !goIdentifierExported(symbol) {
		return "", false
	}
	obj := tc.samePackageGo.Scope().Lookup(symbol)
	if obj == nil {
		return "", false
	}
	if _, ok := obj.(*types.Func); !ok {
		return "", false
	}
	return tc.goHoverMarkdownBodyForResolvedPackage(tc.samePackageGo, tc.samePackageGoImportPath, tc.samePackageGo.Name(), symbol)
}

// GoHoverMarkdownForGoTypeMethod returns hover for a method on a tracked go/types receiver.
func (tc *TypeChecker) GoHoverMarkdownForGoTypeMethod(goType types.Type, importPath, methodName string) (string, bool) {
	if tc == nil || goType == nil || methodName == "" {
		return "", false
	}
	obj, _, _ := types.LookupFieldOrMethod(goType, true, nil, methodName)
	if obj == nil {
		return "", false
	}
	line := types.ObjectString(obj, nil)
	var b strings.Builder
	pkgQual := ""
	if importPath != "" {
		if local, ok := tc.ImportLocalForPath(importPath); ok {
			pkgQual = local
		} else if tc.samePackageGoImportPath == importPath && tc.samePackageGo != nil {
			pkgQual = tc.samePackageGo.Name()
		}
	}
	if pkgQual == "" && importPath != "" {
		parts := strings.Split(importPath, "/")
		pkgQual = parts[len(parts)-1]
	}
	if importPath != "" {
		b.WriteString(tc.goHoverHeader(importPath, pkgQual, methodName))
	} else {
		b.WriteString("**Go method**")
	}
	b.WriteString("\n\n")
	if importPath != "" {
		if docText := tc.goDocParagraphForObject(importPath, obj); docText != "" {
			b.WriteString(docText)
			b.WriteString("\n\n")
		}
	}
	b.WriteString(hoverdoc.GoBlock(line))
	return b.String(), true
}

func (tc *TypeChecker) importPathForGoType(goType types.Type) string {
	if goType == nil {
		return ""
	}
	if ptr, ok := goType.(*types.Pointer); ok {
		goType = ptr.Elem()
	}
	if named, ok := goType.(*types.Named); ok {
		if pkg := named.Obj().Pkg(); pkg != nil {
			return pkg.Path()
		}
	}
	return ""
}

// GoHoverMarkdown returns markdown for hovering an imported Go package (symbol == "") or an exported
// member (symbol set), e.g. pkgLocal "fmt", symbol "Println". Uses go/types when packages loaded.
func (tc *TypeChecker) GoHoverMarkdown(pkgLocal, symbol string) (string, bool) {
	if tc.importPathByLocal == nil {
		return "", false
	}
	path, ok := tc.importPathByLocal[pkgLocal]
	if !ok || path == "" {
		return "", false
	}
	gp := (*types.Package)(nil)
	if tc.goPkgsByLocal != nil {
		gp = tc.goPkgsByLocal[pkgLocal]
	}
	if gp == nil {
		gp = tc.loadedGoPackageByImportPath(path)
	}

	if symbol == "" {
		var b strings.Builder
		b.WriteString("**Go package** `" + pkgLocal + "`")
		if path != "" {
			b.WriteString("\n\n")
			b.WriteString(hoverdoc.GoBlock(fmt.Sprintf("import %q", path)))
		}
		if gp == nil {
			b.WriteString("\n\n*(Go types not loaded — check `go/packages` / `GoWorkspaceDir`.)*")
		}
		return b.String(), true
	}

	if gp == nil {
		if path != "" {
			return fmt.Sprintf("**Go** `%s.%s`\n\n%s\n\n*(Go types not loaded.)*",
				pkgLocal, symbol, hoverdoc.GoBlock(fmt.Sprintf("import %q", path))), true
		}
		return "", false
	}

	return tc.goHoverMarkdownBodyForResolvedPackage(gp, path, pkgLocal, symbol)
}

// GoHoverMarkdownForMethodOnExpression returns hover for expr.method when expr has a tracked Go type
// or a Forst builtin receiver type.
func (tc *TypeChecker) GoHoverMarkdownForMethodOnExpression(recv ast.ExpressionNode, methodName string) (string, bool) {
	if tc == nil || recv == nil || methodName == "" {
		return "", false
	}
	if gt := tc.goTypeForExpression(recv); gt != nil {
		importPath := tc.importPathForGoType(gt)
		return tc.GoHoverMarkdownForGoTypeMethod(gt, importPath, methodName)
	}
	types, err := tc.inferExpressionType(recv)
	if err != nil || len(types) == 0 {
		return "", false
	}
	return tc.GoHoverMarkdownForForstReceiverMethod(types[0], methodName)
}

// GoHoverMarkdownForForstReceiverMethod returns hover for recv.method when the receiver is a Forst
// builtin that maps to a Go type (see forstBuiltinReceiverGoType) and methodName is in that type's
// method set. Documentation text is taken from $GOROOT/src/builtin (package builtin) when
// available; the signature always comes from go/types.
func (tc *TypeChecker) GoHoverMarkdownForForstReceiverMethod(receiverType ast.TypeNode, methodName string) (string, bool) {
	goRecv, goDocName, ok := forstBuiltinReceiverGoType(receiverType)
	if !ok {
		return "", false
	}
	obj, _, _ := types.LookupFieldOrMethod(goRecv, true, nil, methodName)
	if obj == nil {
		return "", false
	}
	line := types.ObjectString(obj, nil)

	p, _ := loadBuiltinGoDocPackage(tc.log)
	docText := builtinGoDocParagraph(p, goDocName, methodName)
	if docText != "" {
		var cdocParser comment.Parser
		var pr comment.Printer
		docText = strings.TrimSpace(string(pr.Markdown(cdocParser.Parse(docText))))
	}

	var b strings.Builder
	b.WriteString("**Go** (predeclared `")
	b.WriteString(goDocName)
	b.WriteString("` — [package builtin](https://pkg.go.dev/builtin))\n\n")
	if docText != "" {
		b.WriteString(docText)
		b.WriteString("\n\n")
	}
	b.WriteString(hoverdoc.GoBlock(line))

	if goDocName == "error" && methodName == "Error" {
		b.WriteString("\n\n")
		b.WriteString(hoverdoc.ForstBlock("Error maps to Go error; Error() returns String"))
	}

	return b.String(), true
}

// GoHoverMarkdownPredeclaredBuiltin returns hover for a bare predeclared Go builtin call name
// (e.g. len, min) when it is registered in BuiltinFunctions with an empty Package.
// Documentation and signatures come from Go's package builtin via go/doc and go/ast.
func (tc *TypeChecker) GoHoverMarkdownPredeclaredBuiltin(name string) (string, bool) {
	if tc == nil || name == "" {
		return "", false
	}
	bfn, ok := BuiltinFunctions[name]
	if !ok || bfn.Package != "" {
		return "", false
	}
	docText, sigGo, ok := builtinGoFuncDoc(tc.log, name)
	if !ok {
		return "", false
	}
	if docText != "" {
		var cdocParser comment.Parser
		var pr comment.Printer
		docText = strings.TrimSpace(string(pr.Markdown(cdocParser.Parse(docText))))
	}

	var b strings.Builder
	b.WriteString("**Go** (predeclared `")
	b.WriteString(name)
	b.WriteString("` — [package builtin](https://pkg.go.dev/builtin))\n\n")
	if docText != "" {
		b.WriteString(docText)
		b.WriteString("\n\n")
	}
	if sigGo != "" {
		b.WriteString(hoverdoc.GoBlock(sigGo))
		b.WriteString("\n\n")
	}
	b.WriteString("**Forst return**\n\n")
	b.WriteString(hoverdoc.ForstBlock(tc.FormatTypeNodeDisplay(bfn.ReturnType)))

	return b.String(), true
}

func mappedForstReturnLines(tc *TypeChecker, mapped []ast.TypeNode) []string {
	parts := make([]string, len(mapped))
	for i, m := range mapped {
		parts[i] = tc.FormatTypeNodeDisplay(m)
	}
	return []string{strings.Join(parts, ", ")}
}

func goSignatureReturnsToForst(sig *types.Signature) []ast.TypeNode {
	res := sig.Results()
	if res.Len() == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}
	}
	out := make([]ast.TypeNode, 0, res.Len())
	for v := range res.Variables() {
		t, ok := goTypeToForstType(v.Type())
		if !ok {
			return nil
		}
		out = append(out, t)
	}
	return out
}

// GoHoverMarkdownForImportPath matches a string literal value from `import "path"` to hover text.
func (tc *TypeChecker) GoHoverMarkdownForImportPath(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	if tc.importPathByLocal != nil {
		for local, p := range tc.importPathByLocal {
			if p == path {
				return tc.GoHoverMarkdown(local, "")
			}
		}
	}
	if gp := tc.loadedGoPackageByImportPath(path); gp != nil {
		var b strings.Builder
		b.WriteString("**Go package** `" + gp.Name() + "`")
		b.WriteString("\n\n")
		b.WriteString(hoverdoc.GoBlock(fmt.Sprintf("import %q", path)))
		return b.String(), true
	}
	return hoverdoc.Section("Go import path") + "\n\n" + hoverdoc.GoBlock(fmt.Sprintf("import %q", path)), true
}
