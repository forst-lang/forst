package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"
	"testing"
)

func TestRecursiveAssertionTypeAlias(t *testing.T) {
	// This test reproduces the bug where assertion types are generated as recursive aliases
	// e.g. type T_5bGqEVyScWp T_5bGqEVyScWp

	// Simulate a value assertion node (e.g. Value("Alice"))
	assertion := &ast.AssertionNode{
		BaseType: nil,
		Constraints: []ast.ConstraintNode{
			{
				Name: "Value",
				Args: []ast.ConstraintArgumentNode{{}},
			},
		},
	}
	typeDef := ast.TypeDefNode{
		Ident: "T_RecursiveAlias",
		Expr: ast.TypeDefAssertionExpr{
			Assertion: assertion,
		},
	}

	log := setupTestLogger()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	decl, err := tr.transformTypeDef(typeDef)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The generated type should NOT be a recursive alias
	// e.g. type T_RecursiveAlias T_RecursiveAlias (which is invalid Go)
	gen, ok := decl.Specs[0].(*goast.TypeSpec)
	if ok && gen.Name.Name == "T_RecursiveAlias" {
		if ident, isIdent := gen.Type.(*goast.Ident); isIdent && ident.Name == gen.Name.Name {
			t.Errorf("TypeDef generated a recursive alias: %v", gen)
		}
	}
	// TODO: Once fixed, assert the correct Go type structure
}

func TestUndefinedTypeForValueAssertion(t *testing.T) {
	// Minimal reproduction: shape with a value assertion field
	assertion := &ast.AssertionNode{
		BaseType:    nil,
		Constraints: []ast.ConstraintNode{{Name: "Value", Args: []ast.ConstraintArgumentNode{{}}}},
	}
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"name": {Assertion: assertion},
		},
	}
	typeDef := ast.TypeDefNode{
		Ident: "T_ShapeWithValue",
		Expr:  ast.TypeDefShapeExpr{Shape: shape},
	}

	log := setupTestLogger()
	tc := setupTypeChecker(log)
	tr := setupTransformer(tc, log)

	decl, err := tr.transformTypeDef(typeDef)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Collect all defined type names from the transformer's output
	defined := map[string]bool{}
	// Add the main type definition
	if gen, ok := decl.Specs[0].(*goast.TypeSpec); ok {
		defined[gen.Name.Name] = true
	}
	// Add all types from the transformer's output
	for _, typeDecl := range tr.Output.types {
		if len(typeDecl.Specs) > 0 {
			if spec, ok := typeDecl.Specs[0].(*goast.TypeSpec); ok {
				defined[spec.Name.Name] = true
			}
		}
	}
	// Collect all referenced type names in struct fields
	referenced := map[string]bool{}
	if gen, ok := decl.Specs[0].(*goast.TypeSpec); ok {
		if structType, ok := gen.Type.(*goast.StructType); ok {
			for _, field := range structType.Fields.List {
				if ident, isIdent := field.Type.(*goast.Ident); isIdent {
					referenced[ident.Name] = true
				}
			}
		}
	}
	// Assert every referenced type is defined and not a recursive alias
	for name := range referenced {
		if !defined[name] {
			t.Errorf("Field references an undefined type: %s", name)
		}
		if name == "T_ShapeWithValue" {
			t.Errorf("TypeDef generated a recursive alias: %s", name)
		}
	}
}

func TestUndefinedTypeForReferencedUserType(t *testing.T) {
	// Minimal reproduction: shape with a field that references a user-defined type
	// This should fail if the referenced type (e.g., AppContext) is not defined

	// First, create the AppContext type definition
	appContextShape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"user": {
				Type: &ast.TypeNode{
					Ident: ast.TypeString,
				},
			},
		},
	}
	appContextDef := ast.TypeDefNode{
		Ident: "AppContext",
		Expr:  ast.TypeDefShapeExpr{Shape: appContextShape},
	}

	// Create a shape that references the user-defined type
	shape := ast.ShapeNode{
		Fields: map[string]ast.ShapeFieldNode{
			"ctx": {
				Type: &ast.TypeNode{
					Ident: ast.TypeIdent("AppContext"),
				},
			},
		},
	}
	typeDef := ast.TypeDefNode{
		Ident: "T_ShapeWithUserType",
		Expr:  ast.TypeDefShapeExpr{Shape: shape},
	}

	log := setupTestLogger()
	tc := setupTypeChecker(log)

	// Register the AppContext type definition in the type checker
	tc.Defs[ast.TypeIdent("AppContext")] = appContextDef

	tr := setupTransformer(tc, log)

	decl, err := tr.transformTypeDef(typeDef)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Collect all defined type names from the transformer's output
	defined := map[string]bool{}
	// Add the main type definition
	if gen, ok := decl.Specs[0].(*goast.TypeSpec); ok {
		defined[gen.Name.Name] = true
	}
	// Add all types from the transformer's output
	for _, typeDecl := range tr.Output.types {
		if len(typeDecl.Specs) > 0 {
			if spec, ok := typeDecl.Specs[0].(*goast.TypeSpec); ok {
				defined[spec.Name.Name] = true
			}
		}
	}

	// Collect all referenced type names in struct fields
	referenced := map[string]bool{}
	if gen, ok := decl.Specs[0].(*goast.TypeSpec); ok {
		if structType, ok := gen.Type.(*goast.StructType); ok {
			for _, field := range structType.Fields.List {
				if ident, isIdent := field.Type.(*goast.Ident); isIdent {
					referenced[ident.Name] = true
				}
			}
		}
	}

	// Assert every referenced type is defined
	for name := range referenced {
		if !defined[name] {
			t.Errorf("Field references an undefined type: %s", name)
		}
	}
}

// TODO: Add tests for:
// 2. Undefined types for value assertions
// 3. Undefined types for referenced user types
// 4. Undefined types for pointer/value assertion types
// 5. Undefined error type
// 6. Invalid Go struct literal in main
