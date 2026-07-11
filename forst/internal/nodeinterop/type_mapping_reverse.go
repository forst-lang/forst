package nodeinterop

import (
	"fmt"
	"sort"
	"strings"

	"forst/internal/ast"
)

// MapIndexTypeToForst maps a forst-index-v1 type node to a Forst TypeNode.
// widened is true when the index type was widened to Object (unknown, empty union, or unmappable union member).
func MapIndexTypeToForst(t IndexType) (ast.TypeNode, bool, error) {
	return indexTypeToForst(t)
}

func indexTypeToForst(t IndexType) (ast.TypeNode, bool, error) {
	switch strings.TrimSpace(t.Kind) {
	case "string":
		return ast.NewBuiltinType(ast.TypeString), false, nil
	case "number":
		return ast.NewBuiltinType(ast.TypeFloat), false, nil
	case "boolean":
		return ast.NewBuiltinType(ast.TypeBool), false, nil
	case "void":
		return ast.NewBuiltinType(ast.TypeVoid), false, nil
	case "bytes":
		return ast.NewBuiltinType(ast.TypeBytes), false, nil
	case "array":
		if t.Element == nil {
			return ast.TypeNode{}, false, fmt.Errorf("array missing element type")
		}
		elem, widened, err := indexTypeToForst(*t.Element)
		if err != nil {
			return ast.TypeNode{}, false, err
		}
		return ast.NewArrayType(elem), widened, nil
	case "object":
		if len(t.Fields) == 0 {
			return ast.NewBuiltinType(ast.TypeObject), true, nil
		}
		ft, err := objectIndexTypeToForst(t.Fields)
		return ft, false, err
	case "union":
		return unionIndexTypeToForst(t.Members)
	case "unknown":
		return ast.NewBuiltinType(ast.TypeObject), true, nil
	default:
		return ast.TypeNode{}, false, fmt.Errorf("unsupported index type kind %q", t.Kind)
	}
}

func unionIndexTypeToForst(members []IndexType) (ast.TypeNode, bool, error) {
	if len(members) == 0 {
		return ast.NewBuiltinType(ast.TypeObject), true, nil
	}
	mapped := make([]ast.TypeNode, 0, len(members))
	widened := false
	for _, member := range members {
		ft, memberWidened, err := indexTypeToForst(member)
		if err != nil {
			return ast.NewBuiltinType(ast.TypeObject), true, nil
		}
		if memberWidened {
			widened = true
		}
		mapped = append(mapped, ft)
	}
	return ast.NewUnionType(mapped...), widened, nil
}

func objectIndexTypeToForst(fields map[string]IndexType) (ast.TypeNode, error) {
	names := make([]string, 0, len(fields))
	for name := range fields {
		if name == "$binary" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	shapeFields := make(map[string]ast.ShapeFieldNode, len(names))
	for _, name := range names {
		fieldType, _, err := indexTypeToForst(fields[name])
		if err != nil {
			return ast.TypeNode{}, fmt.Errorf("field %q: %w", name, err)
		}
		ft := fieldType
		shapeFields[name] = ast.ShapeFieldNode{Type: &ft}
	}

	shape := &ast.ShapeNode{
		Fields:     shapeFields,
		FieldOrder: names,
	}
	assertion := &ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{
			Name: "Shape",
			Args: []ast.ConstraintArgumentNode{{Shape: shape}},
		}},
	}
	return ast.NewAssertionType(assertion), nil
}

// IndexTypeToForst maps a TS index type descriptor to a Forst TypeNode.
func IndexTypeToForst(t IndexType) (ast.TypeNode, error) {
	ft, _, err := MapIndexTypeToForst(t)
	return ft, err
}
