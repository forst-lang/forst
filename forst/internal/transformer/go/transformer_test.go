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

func TestRecursiveTypeAliasBug(t *testing.T) {
	// Test that assertion types are not generated as recursive aliases
	// This test reproduces the bug where assertion types generate invalid Go:
	// type T_5bGqEVyScWp T_5bGqEVyScWp  // Invalid recursive alias

	// The bug is visible in the shape guard example output where we see:
	// type T_5bGqEVyScWp T_5bGqEVyScWp
	// type T_CLULrUt83r1 T_CLULrUt83r1
	//
	// These are invalid Go syntax - types cannot refer to themselves

	// The issue is in the assertion type transformation where value assertions
	// are generating type aliases that reference themselves instead of proper
	// Go types with concrete values

	// For now, this test just documents the bug
	// The actual fix will be implemented in the assertion transformer
	t.Log("Bug documented: assertion types are generated as recursive aliases instead of proper Go types")
}
