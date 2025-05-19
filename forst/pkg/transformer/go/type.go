package transformer_go

import (
	"fmt"
	"forst/pkg/ast"
	goast "go/ast"
)

// transformType converts a Forst type node to a Go type declaration
func (t *Transformer) transformType(n ast.TypeNode) (*goast.Ident, error) {
	switch n.Ident {
	case ast.TypeInt:
		return goast.NewIdent("int"), nil
	case ast.TypeFloat:
		return goast.NewIdent("float64"), nil
	case ast.TypeString:
		return goast.NewIdent("string"), nil
	case ast.TypeBool:
		return goast.NewIdent("bool"), nil
	case ast.TypeVoid:
		return goast.NewIdent("void"), nil
	case ast.TypeError:
		return goast.NewIdent("error"), nil
	case ast.TypeAssertion:
		ident, err := t.TypeChecker.LookupAssertionType(n.Assertion, t.currentScope)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup assertion type: %s", err)
		}
		return goast.NewIdent(string(ident.Ident)), nil
	}
	return nil, fmt.Errorf("unknown type: %s", n.Ident)
}

func (t *Transformer) transformTypes(types []ast.TypeNode) (*goast.FieldList, error) {
	fields := make([]*goast.Field, len(types))
	for i, typ := range types {
		ident, err := t.transformType(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to transform type: %s", err)
		}
		fields[i] = &goast.Field{
			Type: ident,
		}
	}
	return &goast.FieldList{
		List: fields,
	}, nil
}
