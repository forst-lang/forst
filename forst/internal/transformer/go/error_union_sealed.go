package transformergo

import (
	"fmt"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	"go/token"
)

// flattenUnionTypeParams returns leaf members of a TypeUnion tree (same op chain).
func flattenUnionTypeParams(n ast.TypeNode) []ast.TypeNode {
	if n.Ident != ast.TypeUnion || len(n.TypeParams) == 0 {
		return nil
	}
	var out []ast.TypeNode
	for _, m := range n.TypeParams {
		if m.Ident == ast.TypeUnion && len(m.TypeParams) > 0 {
			out = append(out, flattenUnionTypeParams(m)...)
		} else {
			out = append(out, m)
		}
	}
	return out
}

// nominalErrorUnionMembers reports whether every union member is a user-declared nominal error type
// (error X { ... }) and returns their idents. Built-in Error or non-nominal members => false.
func nominalErrorUnionMembers(tc *typechecker.TypeChecker, members []ast.TypeNode) ([]ast.TypeIdent, bool) {
	var out []ast.TypeIdent
	for _, m := range members {
		if m.Ident == "" {
			return nil, false
		}
		if m.Ident == ast.TypeError {
			return nil, false
		}
		def, ok := tc.Defs[m.Ident].(ast.TypeDefNode)
		if !ok {
			return nil, false
		}
		if _, ok := def.Expr.(ast.TypeDefErrorExpr); !ok {
			return nil, false
		}
		out = append(out, m.Ident)
	}
	return out, len(out) >= 2
}

// sealedUnionMethodName returns the unexported interface method used to seal this typedef (e.g. ErrKind -> isErrKind).
func sealedUnionMethodName(typedefName ast.TypeIdent) string {
	s := string(typedefName)
	if len(s) == 0 {
		return "isUnion"
	}
	// is + exported name: ErrKind -> isErrKind
	return "is" + s
}

func (t *Transformer) tryEmitNominalErrorUnionSealedInterface(node ast.TypeDefNode, bin ast.TypeDefBinaryExpr) (*goast.GenDecl, error) {
	canon, err := t.TypeChecker.TypeDefExprToTypeNode(bin)
	if err != nil || canon.Ident != ast.TypeUnion || !t.TypeChecker.IsErrorKindedType(canon) {
		return nil, nil
	}
	members := flattenUnionTypeParams(canon)
	idents, ok := nominalErrorUnionMembers(t.TypeChecker, members)
	if !ok {
		return nil, nil
	}
	methodName := sealedUnionMethodName(node.Ident)
	iface := &goast.InterfaceType{
		Methods: &goast.FieldList{
			List: []*goast.Field{
				{
					Names: []*goast.Ident{goast.NewIdent(methodName)},
					Type: &goast.FuncType{
						Params: &goast.FieldList{},
					},
				},
			},
		},
	}
	typeName := node.Ident
	if typeName == "" {
		return nil, nil
	}
	comments := []*goast.Comment{
		{Text: fmt.Sprintf("// %s is a closed union of nominal errors (only these types implement it).", typeName)},
	}
	decl := &goast.GenDecl{
		Tok: token.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: goast.NewIdent(string(typeName)),
				Type: iface,
			},
		},
		Doc: &goast.CommentGroup{List: comments},
	}
	for _, id := range idents {
		t.emitSealedUnionMemberMethodIfNew(string(id), methodName)
	}
	return decl, nil
}

func (t *Transformer) emitSealedUnionMemberMethodIfNew(receiverType, methodName string) {
	if t.emittedSealMethods == nil {
		t.emittedSealMethods = make(map[string]struct{})
	}
	key := receiverType + "\x00" + methodName
	if _, ok := t.emittedSealMethods[key]; ok {
		return
	}
	t.emittedSealMethods[key] = struct{}{}
	fn := &goast.FuncDecl{
		Recv: &goast.FieldList{
			List: []*goast.Field{
				{Type: goast.NewIdent(receiverType)},
			},
		},
		Name: goast.NewIdent(methodName),
		Type: &goast.FuncType{
			Params: &goast.FieldList{},
		},
		Body: &goast.BlockStmt{},
	}
	t.Output.AddFunction(fn)
}
