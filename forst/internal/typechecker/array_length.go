package typechecker

import (
	"fmt"

	"forst/internal/ast"
)

// arrayLengthsCompatible reports whether two array types have the same slice vs fixed length.
func arrayLengthsCompatible(a, b ast.TypeNode) bool {
	if a.ArrayLen == nil && b.ArrayLen == nil {
		return true
	}
	if a.ArrayLen == nil || b.ArrayLen == nil {
		return false
	}
	return *a.ArrayLen == *b.ArrayLen
}

func intLiteralValue(expr ast.ExpressionNode) (int64, bool) {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		return e.Value, true
	default:
		return 0, false
	}
}

func checkFixedArrayIndexBounds(arrayType ast.TypeNode, index ast.ExpressionNode) error {
	if arrayType.ArrayLen == nil {
		return nil
	}
	idx, ok := intLiteralValue(index)
	if !ok {
		return nil
	}
	if idx < 0 || idx >= *arrayType.ArrayLen {
		return fmt.Errorf("index %d out of range for [%d] array", idx, *arrayType.ArrayLen)
	}
	return nil
}

func checkFixedArrayLiteralLength(arrayType ast.TypeNode, elemCount int) error {
	if arrayType.ArrayLen == nil {
		return nil
	}
	if int64(elemCount) != *arrayType.ArrayLen {
		return fmt.Errorf("array literal has %d elements, want %d for [%d] array", elemCount, *arrayType.ArrayLen, *arrayType.ArrayLen)
	}
	return nil
}
