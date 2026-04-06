package transformergo

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
)

// transformType converts a Forst type node to a Go type declaration
func (t *Transformer) transformType(n ast.TypeNode) (goast.Expr, error) {
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
	case ast.TypePointer:
		if len(n.TypeParams) == 0 {
			return nil, fmt.Errorf("pointer type must have a base type parameter")
		}
		baseType, err := t.transformType(n.TypeParams[0])
		if err != nil {
			return nil, fmt.Errorf("failed to transform pointer base type: %s", err)
		}
		return &goast.StarExpr{X: baseType}, nil
	case ast.TypeArray:
		if len(n.TypeParams) < 1 {
			return nil, fmt.Errorf("array type must have element type parameter")
		}
		elt, err := t.transformType(n.TypeParams[0])
		if err != nil {
			return nil, err
		}
		return &goast.ArrayType{Elt: elt}, nil
	case ast.TypeMap:
		if len(n.TypeParams) < 2 {
			return nil, fmt.Errorf("map type must have key and value type parameters")
		}
		keyT, err := t.transformType(n.TypeParams[0])
		if err != nil {
			return nil, err
		}
		valT, err := t.transformType(n.TypeParams[1])
		if err != nil {
			return nil, err
		}
		return &goast.MapType{Key: keyT, Value: valT}, nil
	default:
		// Always use the unified type aliasing function from the typechecker for all non-builtin, non-special types
		name, err := t.TypeChecker.GetAliasedTypeName(n, typechecker.GetAliasedTypeNameOptions{AllowStructuralAlias: false})
		if err != nil {
			return nil, fmt.Errorf("failed to get aliased type name: %s", err)
		}
		return goast.NewIdent(name), nil
	}
}

func (t *Transformer) transformTypes(types []ast.TypeNode) (*goast.FieldList, error) {
	fields := make([]*goast.Field, len(types))

	for i, typ := range types {
		expr, err := t.transformType(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to transform type: %s", err)
		}
		fields[i] = &goast.Field{
			Type: expr,
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
