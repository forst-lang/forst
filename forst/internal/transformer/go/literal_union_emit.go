package transformergo

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// litKind is the Go carrier kind for a homogeneous literal union member.
type litKind int

const (
	litKindString litKind = iota
	litKindInt
	litKindBool
)

// tryEmitLiteralUnionNamedType emits `type Name string|int|bool`, typed constants, and a
// membership helper `func isName(v carrier) bool` for named homogeneous literal unions.
func (t *Transformer) tryEmitLiteralUnionNamedType(node ast.TypeDefNode) (*goast.GenDecl, error) {
	members, ok := t.TypeChecker.LiteralUnionMembers(ast.TypeNode{
		Ident:    node.Ident,
		TypeKind: ast.TypeKindUserDefined,
	})
	if !ok || len(members) == 0 {
		return nil, nil
	}
	carrierIdent, kind, err := literalUnionCarrierGoIdent(members[0])
	if err != nil {
		return nil, err
	}
	typeName := string(node.Ident)
	var carrierExpr goast.Expr = goast.NewIdent(carrierIdent)

	typeDecl := &goast.GenDecl{
		Tok: token.TYPE,
		Doc: &goast.CommentGroup{List: []*goast.Comment{{
			Text: fmt.Sprintf("// %s: named literal union (%d members)", typeName, len(members)),
		}}},
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: goast.NewIdent(typeName),
				Type: carrierExpr,
			},
		},
	}

	constSpecs := make([]goast.Spec, 0, len(members))
	caseExprs := make([]goast.Expr, 0, len(members))
	seenNames := map[string]int{}
	for _, m := range members {
		constName := literalUnionConstIdent(typeName, m, seenNames)
		litExpr := literalValueToBasicLit(m, kind)
		constSpecs = append(constSpecs, &goast.ValueSpec{
			Names:  []*goast.Ident{goast.NewIdent(constName)},
			Type:   goast.NewIdent(typeName),
			Values: []goast.Expr{litExpr},
		})
		caseExprs = append(caseExprs, goast.NewIdent(constName))
	}
	t.Output.AddValueDecl(&goast.GenDecl{
		Tok:   token.CONST,
		Specs: constSpecs,
	})

	t.Output.AddFunction(&goast.FuncDecl{
		Name: goast.NewIdent(literalUnionMembershipFuncName(typeName)),
		Type: &goast.FuncType{
			Params: &goast.FieldList{List: []*goast.Field{{
				Names: []*goast.Ident{goast.NewIdent("v")},
				Type:  goast.NewIdent(carrierIdent),
			}}},
			Results: &goast.FieldList{List: []*goast.Field{{
				Type: goast.NewIdent("bool"),
			}}},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.SwitchStmt{
				Tag: &goast.CallExpr{
					Fun:  goast.NewIdent(typeName),
					Args: []goast.Expr{goast.NewIdent("v")},
				},
				Body: &goast.BlockStmt{List: []goast.Stmt{
					&goast.CaseClause{
						List: caseExprs,
						Body: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("true")}}},
					},
				}},
			},
			&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("false")}},
		}},
	})

	return typeDecl, nil
}

// literalUnionMembershipFuncName is the Go helper name for membership checks (`isName`).
func literalUnionMembershipFuncName(typeName string) string {
	return "is" + typeName
}

// literalUnionCarrierGoIdent picks the Go carrier type and litKind for a union member.
func literalUnionCarrierGoIdent(v ast.ValueNode) (string, litKind, error) {
	switch v.(type) {
	case ast.StringLiteralNode, *ast.StringLiteralNode:
		return "string", litKindString, nil
	case ast.IntLiteralNode, *ast.IntLiteralNode:
		return "int", litKindInt, nil
	case ast.BoolLiteralNode, *ast.BoolLiteralNode:
		return "bool", litKindBool, nil
	default:
		return "", 0, fmt.Errorf("unsupported literal union member %T", v)
	}
}

// literalValueToBasicLit converts a Forst literal ValueNode into a go/ast basic literal or ident.
func literalValueToBasicLit(v ast.ValueNode, kind litKind) goast.Expr {
	switch kind {
	case litKindString:
		s := ""
		switch x := v.(type) {
		case ast.StringLiteralNode:
			s = x.Value
		case *ast.StringLiteralNode:
			s = x.Value
		}
		return &goast.BasicLit{Kind: token.STRING, Value: strconv.Quote(s)}
	case litKindInt:
		n := int64(0)
		switch x := v.(type) {
		case ast.IntLiteralNode:
			n = x.Value
		case *ast.IntLiteralNode:
			n = x.Value
		}
		return &goast.BasicLit{Kind: token.INT, Value: strconv.FormatInt(n, 10)}
	case litKindBool:
		b := false
		switch x := v.(type) {
		case ast.BoolLiteralNode:
			b = x.Value
		case *ast.BoolLiteralNode:
			b = x.Value
		}
		if b {
			return goast.NewIdent("true")
		}
		return goast.NewIdent("false")
	default:
		return goast.NewIdent("nil")
	}
}

// literalUnionConstIdent builds a unique Go const ident for one literal member under typeName.
func literalUnionConstIdent(typeName string, v ast.ValueNode, seen map[string]int) string {
	raw := "val"
	switch x := v.(type) {
	case ast.StringLiteralNode:
		raw = x.Value
	case *ast.StringLiteralNode:
		raw = x.Value
	case ast.IntLiteralNode:
		raw = strconv.FormatInt(x.Value, 10)
	case *ast.IntLiteralNode:
		raw = strconv.FormatInt(x.Value, 10)
	case ast.BoolLiteralNode:
		raw = strconv.FormatBool(x.Value)
	case *ast.BoolLiteralNode:
		raw = strconv.FormatBool(x.Value)
	}
	sanitized := sanitizeLiteralConstSuffix(raw)
	base := typeName + "_" + sanitized
	if n, ok := seen[base]; ok {
		seen[base] = n + 1
		return fmt.Sprintf("%s_%d", base, n+1)
	}
	seen[base] = 1
	return base
}

// sanitizeLiteralConstSuffix turns a literal text into a safe Go identifier suffix.
func sanitizeLiteralConstSuffix(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteByte('_')
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "x"
	}
	if unicode.IsDigit(rune(out[0])) {
		return "n" + out
	}
	return out
}
