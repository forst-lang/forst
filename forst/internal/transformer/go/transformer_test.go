package transformergo

import (
	"testing"
)

func TestInvalidStructLiteralInMain(t *testing.T) {
	// Test that struct literals in main function are valid Go syntax
	// This test reproduces the bug where struct literals are generated as type declarations
	// instead of initialized struct literals

	// The bug is visible in the shape guard example output where we see:
	// op := struct {
	//     ctx   T_hBjKiyPmxT
	//     input T_DK4A5vCj4Wc
	// }
	// instead of:
	// op := struct {
	//     ctx   T_hBjKiyPmxT
	//     input T_DK4A5vCj4Wc
	// }{
	//     ctx:   T_hBjKiyPmxT{...},
	//     input: T_DK4A5vCj4Wc{...},
	// }

	// The issue is in transformExpression when it encounters a ShapeNode
	// It calls transformShapeType which generates a type declaration instead of
	// a struct literal with initialization values

	// For now, this test just documents the bug
	// The actual fix will be implemented in the expression transformer
	t.Log("Bug documented: struct literals in main function are generated as type declarations instead of initialized struct literals")
}
