package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/parser"
	"forst/internal/typechecker"
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

func TestUndefinedUserTypesBug(t *testing.T) {
	// Test that user-defined types are properly emitted in the output
	// This test reproduces the bug where user-defined types are referenced
	// but not defined in the generated Go code

	// The bug is visible in the shape guard example output where we see:
	// undefined: AppContext
	// undefined: T_488eVThFocF
	//
	// These are user-defined types that are referenced but not emitted

	// The issue is in the type emission logic where some user-defined types
	// are not being added to the output, causing compilation errors

	// For now, this test just documents the bug
	// The actual fix will be implemented in the type emission logic
	t.Log("Bug documented: user-defined types are referenced but not emitted in generated Go code")
}

func TestMissingReturnValuesBug(t *testing.T) {
	// Test that functions return the expected values
	// This test reproduces the bug where functions are missing return values

	// The bug is visible in the shape guard example output where we see:
	// not enough return values
	//   have (error)
	//   want (string, error)
	//
	// The createUser function is missing the string return value

	// The issue is in the function transformation logic where return
	// statements are not being properly transformed to return all
	// expected values

	// For now, this test just documents the bug
	// The actual fix will be implemented in the function transformer
	t.Log("Bug documented: functions are missing return values in generated Go code")
}

func TestDeterministicShapeGuardExample(t *testing.T) {
	// Load the shape guard example source
	inputPath := filepath.Join("..", "..", "..", "..", "examples", "in", "rfc", "guard", "shape_guard.ft")
	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("Failed to read shape guard example: %v", err)
	}
	input := string(inputBytes)

	var lastOutput []byte
	for i := 0; i < 100; i++ {
		// Parse
		p := parser.NewTestParser(input)
		astNodes, err := p.ParseFile()
		if err != nil {
			t.Fatalf("Parse failed on run %d: %v", i, err)
		}
		// Typecheck
		tc := typechecker.New(nil, false)
		err = tc.CheckTypes(astNodes)
		if err != nil {
			t.Fatalf("Typecheck failed on run %d: %v", i, err)
		}
		// Transform
		tr := New(tc, nil)
		goFile, err := tr.TransformForstFileToGo(astNodes)
		if err != nil {
			t.Fatalf("Transform failed on run %d: %v", i, err)
		}
		// Render Go code
		var buf bytes.Buffer
		if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
			t.Fatalf("Go formatting failed on run %d: %v", i, err)
		}
		output := buf.Bytes()
		if i > 0 && !bytes.Equal(output, lastOutput) {
			tmp0 := filepath.Join(os.TempDir(), "determinism_fail_run0.go")
			tmpN := filepath.Join(os.TempDir(), "determinism_fail_runN.go")
			os.WriteFile(tmp0, lastOutput, 0644)
			os.WriteFile(tmpN, output, 0644)
			t.Fatalf("Compiler output is not deterministic on run %d. See %s and %s for diff.", i, tmp0, tmpN)
		}
		lastOutput = output
	}
}
