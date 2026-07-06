package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
)

func TestUnifyTypeguard_rejectsNominalErrorBareIsGuard(t *testing.T) {
	src := `package main

error ParseError { code: Int }

func main() {
	x := mk()
	if x is ParseError {
		println(x)
	}
}

func mk(): Result(Int, ParseError) {
	return 0
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{UseModuleRoot: true})
	if err == nil {
		t.Fatal("expected error for bare nominal error is guard")
	}
	if !strings.Contains(err.Error(), "nominal error type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnifyTypeguard_presentRequiresPointer(t *testing.T) {
	src := `package main

func main() {
	n := 1
	if n is Present() {
		println(n)
	}
}
`
	_, _, err := Typecheck(t, src, testutil.TypecheckOpts{})
	if err == nil {
		t.Fatal("expected error for Present on non-pointer")
	}
	if !strings.Contains(err.Error(), "present assertion requires a pointer") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnifyTypeguard_okDiscriminatorOnResult(t *testing.T) {
	src := `package main

func main() {
	x := mk()
	if x is Ok() {
		println(x)
	}
}

func mk(): Result(Int, String) {
	return 0
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestUnifyTypeguard_typeGuardSubjectMismatch(t *testing.T) {
	tc := New(setupTestLogger(nil), false)
	tc.Defs["Positive"] = ast.TypeGuardNode{
		Ident: ast.Identifier("Positive"),
		Subject: ast.SimpleParamNode{
			Ident: ast.Ident{ID: "n"},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		},
	}
	err := tc.validateAssertionNode(ast.AssertionNode{
		Constraints: []ast.ConstraintNode{{Name: "Positive"}},
	}, ast.TypeNode{Ident: ast.TypeString})
	if err == nil {
		t.Fatal("expected type guard subject mismatch error")
	}
	if !strings.Contains(err.Error(), "type guard 'Positive' requires subject type Int") {
		t.Fatalf("unexpected error: %v", err)
	}
}
