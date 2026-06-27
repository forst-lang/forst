package transformergo

import (
	"fmt"
	"strings"

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
	case ast.TypeChannel:
		if len(n.TypeParams) < 1 {
			return nil, fmt.Errorf("channel type must have element type parameter")
		}
		elt, err := t.transformType(n.TypeParams[0])
		if err != nil {
			return nil, err
		}
		return &goast.ChanType{
			Dir:   goast.SEND | goast.RECV,
			Value: elt,
		}, nil
	case ast.TypeResult:
		return nil, fmt.Errorf("result types are expanded at function boundaries; use transformTypes, not transformType, or transformResultAsStructFieldGoType for struct fields")
	case ast.TypeTuple:
		return nil, fmt.Errorf("tuple types are expanded at function boundaries; use transformTypes, not transformType")
	case ast.TypeUnion:
		if t.TypeChecker.IsErrorKindedType(n) {
			var r goast.Expr = goast.NewIdent("error")
			return r, nil
		}
		var r goast.Expr = goast.NewIdent("any")
		return r, nil
	case ast.TypeIntersection:
		var r goast.Expr = goast.NewIdent("any")
		return r, nil
	default:
		id := string(n.Ident)
		if strings.Contains(id, ".") {
			parts := strings.Split(id, ".")
			if len(parts) == 2 {
				return &goast.SelectorExpr{X: goast.NewIdent(parts[0]), Sel: goast.NewIdent(parts[1])}, nil
			}
		}
		// Always use the unified type aliasing function from the typechecker for all non-builtin, non-special types
		name, err := t.TypeChecker.GetAliasedTypeName(n, typechecker.GetAliasedTypeNameOptions{AllowStructuralAlias: false})
		if err != nil {
			return nil, fmt.Errorf("failed to get aliased type name: %s", err)
		}
		return goast.NewIdent(name), nil
	}
}

func (t *Transformer) transformTypes(types []ast.TypeNode) (*goast.FieldList, error) {
	var fields []*goast.Field
	for _, typ := range types {
		if typ.IsResultType() {
			if len(typ.TypeParams) != 2 {
				return nil, fmt.Errorf("result must have exactly two type parameters")
			}
			s, err := t.transformType(typ.TypeParams[0])
			if err != nil {
				return nil, fmt.Errorf("failed to transform Result success type: %w", err)
			}
			fields = append(fields, &goast.Field{Type: s})
			fields = append(fields, &goast.Field{Type: goast.NewIdent("error")})
			continue
		}
		if typ.IsTupleType() {
			for _, elem := range typ.TypeParams {
				expr, err := t.transformType(elem)
				if err != nil {
					return nil, fmt.Errorf("failed to transform Tuple element: %w", err)
				}
				fields = append(fields, &goast.Field{Type: expr})
			}
			continue
		}
		expr, err := t.transformType(typ)
		if err != nil {
			return nil, fmt.Errorf("failed to transform type: %s", err)
		}
		fields = append(fields, &goast.Field{Type: expr})
	}

	return &goast.FieldList{
		List: fields,
	}, nil
}

// transformResultAsStructFieldGoType lowers Result(S, F) stored in a Forst shape field to a single
// Go struct type { V S; Err F }. Call sites must use the same layout for literals and field access
// (.V success payload, .Err failure / error slot). Only failure type Error is supported for now
// (matches Go error checks on .Err).
func (t *Transformer) transformResultAsStructFieldGoType(rt ast.TypeNode) (*goast.StructType, error) {
	if !rt.IsResultType() || len(rt.TypeParams) < 2 {
		return nil, fmt.Errorf("expected result(S, F)")
	}
	fail := rt.TypeParams[1]
	if fail.Ident != ast.TypeError {
		return nil, fmt.Errorf("result in struct fields: failure type must be Error for Go codegen (got %s)", fail.String())
	}
	s, err := t.transformType(rt.TypeParams[0])
	if err != nil {
		return nil, fmt.Errorf("result field success type: %w", err)
	}
	e, err := t.transformType(fail)
	if err != nil {
		return nil, fmt.Errorf("result field failure type: %w", err)
	}
	return &goast.StructType{
		Fields: &goast.FieldList{
			List: []*goast.Field{
				{Names: []*goast.Ident{goast.NewIdent(loweredResultValueFieldName)}, Type: s},
				{Names: []*goast.Ident{goast.NewIdent(loweredResultErrFieldName)}, Type: e},
			},
		},
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
