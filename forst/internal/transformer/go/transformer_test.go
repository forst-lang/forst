package transformergo

import (
	"bytes"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
	"os/exec"

	"github.com/sirupsen/logrus"
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
		logger := ast.SetupTestLogger()
		p := parser.NewTestParser(input, logger)
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

func TestHashBasedTypeGenerationIssue(t *testing.T) {
	// This test reproduces the exact issue where the transformer generates
	// hash-based types (e.g., T_TndA4HRxw7P) that are undefined in the output
	// and causes type mismatches in function signatures and return statements.

	// The issue manifests as:
	// 1. undefined: T_TndA4HRxw7P
	// 2. cannot use User{} (value of struct type User) as CreateUserResponse value
	// 3. not enough return values

	input := `
package user

type User = {
  id: String,
  name: String,
  age: Int,
  email: String
}

type CreateUserRequest = {
  name: String,
  age: Int,
  email: String
}

type CreateUserResponse = {
  user: User,
  created_at: Int
}

func CreateUser(input CreateUserRequest) {
  user := {
    id: "123",
    name: input.name,
    age: input.age,
    email: input.email
  }
  
  return {
    user: user,
    created_at: 1234567890
  }
}
`

	// Parse
	logger := ast.SetupTestLogger()
	if testing.Verbose() {
		logger.SetLevel(logrus.DebugLevel)
	}
	p := parser.NewTestParser(input, logger)
	astNodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Typecheck
	tc := typechecker.New(nil, false)
	err = tc.CheckTypes(astNodes)
	if err != nil {
		t.Fatalf("Typecheck failed: %v", err)
	}

	// Debug: Print available types
	t.Logf("Available types in typechecker:")
	for typeIdent, def := range tc.Defs {
		t.Logf("  %s: %T", typeIdent, def)
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			if shapeExpr, ok := typeDef.Expr.(ast.TypeDefShapeExpr); ok {
				t.Logf("    Shape fields: %+v", shapeExpr.Shape.Fields)
			}
		}
	}

	// Debug: Create a test shape to see what the matching logic finds
	testShape := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id":    {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"name":  {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"age":   {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			"email": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	t.Logf("Test shape fields: %+v", testShape.Fields)

	// Test the shape matching logic directly
	tr := New(tc, nil)

	// Debug: Test the getExpectedTypeForShape function directly
	t.Logf("Testing getExpectedTypeForShape directly:")
	testShapeForDebug := &ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"id":    {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"name":  {Type: &ast.TypeNode{Ident: ast.TypeString}},
			"age":   {Type: &ast.TypeNode{Ident: ast.TypeInt}},
			"email": {Type: &ast.TypeNode{Ident: ast.TypeString}},
		},
	}
	context := &ShapeContext{
		VariableName: "user",
	}
	expectedType := tr.getExpectedTypeForShape(testShapeForDebug, context)
	t.Logf("getExpectedTypeForShape result: %+v", expectedType)
	if expectedType != nil {
		t.Logf("Expected type ident: %s", expectedType.Ident)
	}
	typeIdent, found := tr.findExistingTypeForShape(testShape, nil)
	t.Logf("Shape matching result: found=%v, typeIdent=%s", found, typeIdent)

	// Transform
	goFile, err := tr.TransformForstFileToGo(astNodes)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	// Render Go code
	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), goFile); err != nil {
		t.Fatalf("Go formatting failed: %v", err)
	}
	output := buf.String()

	// Check for the specific issues
	if !strings.Contains(output, "type User struct") {
		t.Error("Expected User type definition to be emitted")
	}

	if !strings.Contains(output, "type CreateUserRequest struct") {
		t.Error("Expected CreateUserRequest type definition to be emitted")
	}

	if !strings.Contains(output, "type CreateUserResponse struct") {
		t.Error("Expected CreateUserResponse type definition to be emitted")
	}

	// Check for hash-based types that shouldn't be there
	if strings.Contains(output, "T_") {
		t.Errorf("Generated code contains hash-based types: %s", output)
	}

	// Check that function signature is correct
	if !strings.Contains(output, "func CreateUser(input CreateUserRequest) CreateUserResponse {") {
		t.Error("Expected correct function signature with CreateUserRequest parameter and CreateUserResponse return")
	}

	// Check that return statement uses correct type
	if !strings.Contains(output, "return CreateUserResponse{") {
		t.Error("Expected return statement to use CreateUserResponse type")
	}

	// Verify the generated code compiles
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "user.go")
	if err := os.WriteFile(testFile, []byte(output), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "user"), testFile)
	if buildOutput, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("Generated Go code does not compile:\n%s\n\nGenerated code:\n%s", string(buildOutput), output)
	}
}
