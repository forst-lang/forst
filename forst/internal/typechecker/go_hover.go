package typechecker

import (
	"fmt"
	"go/doc/comment"
	"go/types"
	"strings"

	"forst/internal/ast"
)

// IsImportedLocalName reports whether id is a Go import's local identifier in this file.
func (tc *TypeChecker) IsImportedLocalName(id string) bool {
	if tc.importPathByLocal == nil {
		return false
	}
	_, ok := tc.importPathByLocal[id]
	return ok
}

// loadedGoPackageByImportPath returns a loaded *types.Package for the given import path, including
// dot-imported packages (which are not keyed in goPkgsByLocal).
func (tc *TypeChecker) loadedGoPackageByImportPath(path string) *types.Package {
	if path == "" {
		return nil
	}
	if tc.importPathByLocal != nil && tc.goPkgsByLocal != nil {
		for local, p := range tc.importPathByLocal {
			if p == path {
				if gp := tc.goPkgsByLocal[local]; gp != nil {
					return gp
				}
			}
		}
	}
	for _, dp := range tc.dotImportPkgs {
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

func (tc *TypeChecker) goHoverMarkdownBodyForResolvedPackage(gp *types.Package, pkgQual, symbol string) (string, bool) {
	obj := gp.Scope().Lookup(symbol)
	if obj == nil {
		return fmt.Sprintf("**Go** `%s.%s` — not found in `%s`", pkgQual, symbol, gp.Path()), true
	}

	qf := types.RelativeTo(gp)
	head := types.ObjectString(obj, qf)
	var b strings.Builder
	b.WriteString("```go\n" + head + "\n```")

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
			if mapped := goSignatureReturnsToForst(sig); len(mapped) > 0 {
				b.WriteString("\n\n**Forst-mapped returns** ")
				for i, m := range mapped {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString("`" + tc.FormatTypeNodeDisplay(m) + "`")
				}
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
	return tc.goHoverMarkdownBodyForResolvedPackage(gp, gp.Name(), symbol)
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
			fmt.Fprintf(&b, "\n\n```go\nimport %q\n```", path)
		}
		if gp == nil {
			b.WriteString("\n\n*(Go types not loaded — check `go/packages` / `GoWorkspaceDir`.)*")
		}
		return b.String(), true
	}

	if gp == nil {
		if path != "" {
			return fmt.Sprintf("**Go** `%s.%s`\n\n```go\nimport %q\n```\n\n*(Go types not loaded.)*", pkgLocal, symbol, path), true
		}
		return "", false
	}

	return tc.goHoverMarkdownBodyForResolvedPackage(gp, pkgLocal, symbol)
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
	b.WriteString("```go\n")
	b.WriteString(line)
	b.WriteString("\n```")

	if goDocName == "error" && methodName == "Error" {
		b.WriteString("\n\n**Forst** `Error` maps to Go `error`; `Error()` returns `String`.")
	}

	return b.String(), true
}

func goSignatureReturnsToForst(sig *types.Signature) []ast.TypeNode {
	res := sig.Results()
	if res.Len() == 0 {
		return []ast.TypeNode{{Ident: ast.TypeVoid}}
	}
	out := make([]ast.TypeNode, 0, res.Len())
	for i := 0; i < res.Len(); i++ {
		t, ok := goTypeToForstType(res.At(i).Type())
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
		fmt.Fprintf(&b, "\n\n```go\nimport %q\n```", path)
		return b.String(), true
	}
	return fmt.Sprintf("**Go import path** `%s`", path), true
}
