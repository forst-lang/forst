package typechecker

import (
	"strings"

	"forst/internal/ast"
)

// FieldHoverMarkdown returns markdown for a dotted field path rooted at a variable (e.g. move.state
// or move.state.status). Uses lookupFieldPath; ok is false if the path cannot be resolved.
// parentTypeForDoc is the named type whose shape defines the last field (for leading comments), when known.
func (tc *TypeChecker) FieldHoverMarkdown(root ast.Identifier, span ast.SourceSpan, fieldPath []string) (md string, parentTypeForDoc ast.TypeIdent, ok bool) {
	if tc == nil || len(fieldPath) == 0 {
		return "", "", false
	}
	var baseTypes []ast.TypeNode
	var have bool
	if span.IsSet() {
		vn := ast.VariableNode{Ident: ast.Ident{ID: root, Span: span}}
		baseTypes, have = tc.InferredTypesForVariableNode(vn)
	}
	if !have || len(baseTypes) == 0 {
		baseTypes, have = tc.InferredTypesForVariableIdentifier(root)
	}
	if !have || len(baseTypes) == 0 {
		return "", "", false
	}
	last := fieldPath[len(fieldPath)-1]
	displayPath := string(root) + "." + strings.Join(fieldPath, ".")
	for _, bt := range baseTypes {
		resolved, err := tc.lookupFieldPath(bt, fieldPath)
		if err != nil {
			continue
		}
		typeStr := tc.FormatTypeNodeDisplay(resolved)
		pid, _ := tc.ParentTypeIdentForFieldPath(bt, fieldPath)
		var b strings.Builder
		b.WriteString("**`")
		b.WriteString(last)
		b.WriteString("`** (field)\n\n")
		b.WriteString("```forst\n")
		b.WriteString(displayPath)
		b.WriteString(": ")
		b.WriteString(typeStr)
		b.WriteString("\n```")
		return b.String(), pid, true
	}
	return "", "", false
}

// ParentTypeIdentForFieldPath returns the named type whose shape defines the last segment of
// fieldPath (for documentation lookup). For move.state.status with path [state, status], the parent
// of "status" is the type after resolving [state] (e.g. GameState).
func (tc *TypeChecker) ParentTypeIdentForFieldPath(base ast.TypeNode, fieldPath []string) (ast.TypeIdent, bool) {
	if tc == nil || len(fieldPath) == 0 {
		return "", false
	}
	parentPath := fieldPath[:len(fieldPath)-1]
	var parentT ast.TypeNode
	var err error
	if len(parentPath) == 0 {
		parentT = base
	} else {
		parentT, err = tc.lookupFieldPath(base, parentPath)
		if err != nil {
			return "", false
		}
	}
	p := tc.GetMostSpecificNonHashAlias(parentT)
	if p.Ident == "" {
		return "", false
	}
	return ast.TypeIdent(p.Ident), true
}

// ShapeFieldFromTypeDef returns the shape field node for a named type's field, if the definition is a shape.
func (tc *TypeChecker) ShapeFieldFromTypeDef(typeName ast.TypeIdent, fieldName string) (ast.ShapeFieldNode, bool) {
	if tc == nil {
		return ast.ShapeFieldNode{}, false
	}
	def, ok := tc.Defs[typeName]
	if !ok {
		return ast.ShapeFieldNode{}, false
	}
	td, ok := def.(ast.TypeDefNode)
	if !ok {
		return ast.ShapeFieldNode{}, false
	}
	se, ok := td.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return ast.ShapeFieldNode{}, false
	}
	f, ok := se.Shape.Fields[fieldName]
	return f, ok
}
