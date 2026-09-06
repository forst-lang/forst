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
		return reportf(spanOfExpression(index), "index-out-of-range",
			fmt.Sprintf("index %d out of range", idx),
			fmt.Sprintf("Index %d is out of range for a [%d] array.", idx, *arrayType.ArrayLen),
			fmt.Sprintf("use an index from 0 to %d", *arrayType.ArrayLen-1))
	}
	return nil
}

func checkFixedArrayLiteralLength(arrayType ast.TypeNode, elemCount int, span ast.SourceSpan) error {
	if arrayType.ArrayLen == nil {
		return nil
	}
	if int64(elemCount) != *arrayType.ArrayLen {
		return reportf(span, "array-length-mismatch",
			"array literal length mismatch",
			fmt.Sprintf("Array literal has %d elements, want %d for [%d] array.", elemCount, *arrayType.ArrayLen, *arrayType.ArrayLen),
			fmt.Sprintf("provide exactly %d elements", *arrayType.ArrayLen))
	}
	return nil
}
