package typechecker

import (
	"fmt"
	"forst/internal/ast"

	"go/types"
)

func (tc *TypeChecker) inferExpressionIndex(expr ast.Node) ([]ast.TypeNode, bool, error) {
	e, ok := expr.(ast.IndexExpressionNode)
	if !ok {
		return nil, false, nil
	}
	targetTypes, err := tc.inferExpressionType(e.Target)
	if err != nil {
		return nil, true, err
	}
	if len(targetTypes) != 1 {
		return nil, true, reportf(spanIndexExpr(e), "index-target-type",
			"index target must have a single type",
			fmt.Sprintf("The indexed value has %d types; subscript needs exactly one.", len(targetTypes)),
			"bind the container to a single-typed name")
	}
	indexTypes, err := tc.inferExpressionType(e.Index)
	if err != nil {
		return nil, true, err
	}
	if len(indexTypes) != 1 {
		return nil, true, reportf(spanIndexExpr(e), "index-type",
			"index expression must have a single index type",
			"The index expression must infer to exactly one type.",
			"use an Int index for slices/arrays/strings or the map key type")
	}
	t := targetTypes[0]
	switch {
	case t.Ident == ast.TypeMap && len(t.TypeParams) >= 2:
		return tc.inferMapIndexExpressionType(e, t, indexTypes[0])
	case t.Ident == ast.TypeString:
		return tc.inferStringIndexExpressionType(e, indexTypes[0])
	case t.Ident == ast.TypeBytes:
		return tc.inferBytesIndexExpressionType(e, indexTypes[0])
	default:
		return tc.inferArrayIndexExpressionType(e, t, indexTypes[0])
	}
}

func (tc *TypeChecker) inferMapIndexExpressionType(e ast.IndexExpressionNode, container ast.TypeNode, index ast.TypeNode) ([]ast.TypeNode, bool, error) {
	wantK, wantV := container.TypeParams[0], container.TypeParams[1]
	if !tc.IsTypeCompatible(index, wantK) {
		return nil, true, reportf(spanIndexExpr(e), "index-type",
			"map index key type mismatch",
			fmt.Sprintf("Map key type must be `%s`, got `%s`.", formatTypeIdentForDiag(wantK.Ident), formatTypeIdentForDiag(index.Ident)),
			"convert the key or change the map's key type")
	}
	resultType := ast.TypeNode{
		Ident: ast.TypeResult,
		TypeParams: []ast.TypeNode{
			wantV,
			{Ident: ast.TypeError},
		},
	}
	tc.storeInferredType(e, []ast.TypeNode{resultType})
	return []ast.TypeNode{resultType}, true, nil
}

func (tc *TypeChecker) inferStringIndexExpressionType(e ast.IndexExpressionNode, index ast.TypeNode) ([]ast.TypeNode, bool, error) {
	if index.Ident != ast.TypeInt {
		return nil, true, reportf(spanIndexExpr(e), "index-type",
			"string index must be Int",
			fmt.Sprintf("String indexing requires Int, got `%s`.", index.Ident),
			"use an integer index")
	}
	elem := ast.TypeNode{Ident: ast.TypeInt}
	tc.storeInferredType(e, []ast.TypeNode{elem})
	return []ast.TypeNode{elem}, true, nil
}

func (tc *TypeChecker) inferBytesIndexExpressionType(e ast.IndexExpressionNode, index ast.TypeNode) ([]ast.TypeNode, bool, error) {
	if index.Ident != ast.TypeInt {
		return nil, true, reportf(spanIndexExpr(e), "index-type",
			"[]byte index must be Int",
			fmt.Sprintf("`[]byte` indexing requires Int, got `%s`.", index.Ident),
			"use an integer index")
	}
	elem := ast.TypeNode{Ident: ast.TypeIdent("byte")}
	tc.storeInferredType(e, []ast.TypeNode{elem})
	return []ast.TypeNode{elem}, true, nil
}

func (tc *TypeChecker) inferArrayIndexExpressionType(e ast.IndexExpressionNode, container ast.TypeNode, index ast.TypeNode) ([]ast.TypeNode, bool, error) {
	if container.Ident != ast.TypeArray || len(container.TypeParams) < 1 {
		return nil, true, reportf(spanIndexExpr(e), "index-target-type",
			"index target must be map, slice, or array",
			fmt.Sprintf("Cannot index type `%s`; expected map, slice, array, string, or []byte.", formatTypeIdentForDiag(container.Ident)),
			"index a supported container type")
	}
	if index.Ident != ast.TypeInt {
		return nil, true, reportf(spanIndexExpr(e), "index-type",
			"slice/array index must be Int",
			fmt.Sprintf("Slice and array indexes must be Int, got `%s`.", index.Ident),
			"use an integer index")
	}
	if err := checkFixedArrayIndexBounds(container, e.Index); err != nil {
		return nil, true, err
	}
	elem := container.TypeParams[0]
	tc.storeInferredType(e, []ast.TypeNode{elem})
	return []ast.TypeNode{elem}, true, nil
}

func (tc *TypeChecker) inferExpressionSlice(expr ast.Node) ([]ast.TypeNode, bool, error) {
	e, ok := expr.(ast.SliceExpressionNode)
	if !ok {
		return nil, false, nil
	}
	if goT := tc.goTypeForExpression(e.Target); goT != nil {
		return tc.inferGoSliceExpressionType(e, goT)
	}
	targetTypes, err := tc.inferExpressionType(e.Target)
	if err != nil {
		return nil, true, err
	}
	if len(targetTypes) != 1 || targetTypes[0].Ident != ast.TypeArray || len(targetTypes[0].TypeParams) < 1 {
		return nil, true, reportf(spanSliceExpr(e), "slice-target-type",
			"slice target must be a slice or array",
			"The slice target must be a Forst slice or array type.",
			"slice a []T or [N]T value")
	}
	if err := tc.checkSliceBounds(e); err != nil {
		return nil, true, err
	}
	elem := targetTypes[0].TypeParams[0]
	out := ast.TypeNode{Ident: ast.TypeArray, TypeParams: []ast.TypeNode{elem}}
	tc.storeInferredType(e, []ast.TypeNode{out})
	return []ast.TypeNode{out}, true, nil
}

func (tc *TypeChecker) inferGoSliceExpressionType(e ast.SliceExpressionNode, goT types.Type) ([]ast.TypeNode, bool, error) {
	var elem types.Type
	switch u := goT.Underlying().(type) {
	case *types.Slice:
		elem = u.Elem()
	case *types.Array:
		elem = u.Elem()
	default:
		return nil, true, reportf(spanSliceExpr(e), "slice-target-type",
			"slice target must be a slice or array",
			fmt.Sprintf("Cannot slice type `%s`; expected a slice or array.", goT.String()),
			"slice a []T or [N]T value")
	}
	if err := tc.checkSliceBounds(e); err != nil {
		return nil, true, err
	}
	ft, ok := tc.mapGoType(types.NewSlice(elem))
	if !ok {
		return nil, true, reportf(spanSliceExpr(e), "slice-target-type",
			"cannot map Go slice element type",
			"The Go slice element type could not be mapped to a Forst type.",
			"slice a supported Go slice or array type")
	}
	tc.storeInferredType(e, []ast.TypeNode{ft})
	return []ast.TypeNode{ft}, true, nil
}

func (tc *TypeChecker) checkSliceBounds(e ast.SliceExpressionNode) error {
	if e.Low != nil {
		lowTypes, err := tc.inferExpressionType(e.Low)
		if err != nil {
			return err
		}
		if len(lowTypes) != 1 || lowTypes[0].Ident != ast.TypeInt {
			return reportf(spanSliceExpr(e), "slice-bound",
				"slice low bound must be Int",
				"The low bound of a slice expression must be Int.",
				"use an integer low bound")
		}
	}
	if e.High != nil {
		highTypes, err := tc.inferExpressionType(e.High)
		if err != nil {
			return err
		}
		if len(highTypes) != 1 || highTypes[0].Ident != ast.TypeInt {
			return reportf(spanSliceExpr(e), "slice-bound",
				"slice high bound must be Int",
				"The high bound of a slice expression must be Int.",
				"use an integer high bound")
		}
	}
	return nil
}

func (tc *TypeChecker) inferExpressionSpread(expr ast.Node) ([]ast.TypeNode, bool, error) {
	e, ok := expr.(ast.SpreadExpressionNode)
	if !ok {
		return nil, false, nil
	}
	ts, err := tc.inferExpressionType(e.Expr)
	if err != nil {
		return nil, true, err
	}
	tc.storeInferredType(e, ts)
	return ts, true, nil
}
