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

	obj := gp.Scope().Lookup(symbol)
	if obj == nil {
		if path != "" {
			return fmt.Sprintf("**Go** `%s.%s` — not found in `%s`", pkgLocal, symbol, path), true
		}
		return "", false
	}

	qf := types.RelativeTo(gp)
	head := types.ObjectString(obj, qf)
	var b strings.Builder
	b.WriteString("```go\n" + head + "\n```")

	qual := pkgLocal + "." + symbol
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
	for local, p := range tc.importPathByLocal {
		if p == path {
			return tc.GoHoverMarkdown(local, "")
		}
	}
	return fmt.Sprintf("**Go import path** `%s`", path), true
}
