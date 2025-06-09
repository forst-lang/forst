package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
)

// transformType converts a Forst type node to a Go type declaration
func (t *Transformer) transformType(n ast.TypeNode) (*goast.Ident, error) {
	if n.Ident == "" {
		return nil, fmt.Errorf("TypeNode is missing an identifier: %+v", n)
	}
	switch n.Ident {
	case ast.TypeAssertion:
		ident, err := t.TypeChecker.LookupAssertionType(n.Assertion)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup assertion type: %s", err)
		}
		return goast.NewIdent(string(ident.Ident)), nil
	case ast.TypeImplicit:
		return nil, fmt.Errorf("TypeImplicit is not a valid Go type")
	case ast.TypeObject:
		return nil, fmt.Errorf("TypeObject should not be used as a Go type")
	default:
		ident, err := transformTypeIdent(n.Ident)
		if err != nil {
			return nil, fmt.Errorf("failed to transform type ident: %s", err)
		}
		return ident, nil
	}
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

func transformTypeIdent(ident ast.TypeIdent) (*goast.Ident, error) {
	switch ident {
	case ast.TypeString:
		return &goast.Ident{Name: "string"}, nil
	case ast.TypeInt:
		return &goast.Ident{Name: "int"}, nil
	case ast.TypeFloat:
		return &goast.Ident{Name: "float64"}, nil
	case ast.TypeBool:
		return &goast.Ident{Name: "bool"}, nil
	case ast.TypeVoid:
		return &goast.Ident{Name: "void"}, nil
	case ast.TypeError:
		return &goast.Ident{Name: "error"}, nil
	case ast.TypeObject:
		return nil, fmt.Errorf("TypeObject should not be used as a Go type")
	case ast.TypeAssertion:
		return nil, fmt.Errorf("TypeAssertion should not be used as a Go type")
	case ast.TypeImplicit:
		return nil, fmt.Errorf("TypeImplicit should not be used as a Go type")
	default:
		// For user-defined types (aliases, shapes, etc.), just use the type name
		return goast.NewIdent(string(ident)), nil
	}
}
