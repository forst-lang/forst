package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestValidateDeferGoBuiltinRestriction_specDenylist(t *testing.T) {
	t.Parallel()
	tests := []struct {
		callee string
		wantErr bool
	}{
		{"len", true},
		{"append", true},
		{"make", true},
		{"new", true},
		{"cap", true},
		{"complex", true},
		{"imag", true},
		{"real", true},
		{"unsafe.Add", true},
		{"unsafe.Sizeof", true},
		{"println", false},
		{"close", false},
		{"panic", false},
		{"fmt.Println", false},
	}
	for _, tt := range tests {
		t.Run(tt.callee, func(t *testing.T) {
			t.Parallel()
			call := ast.FunctionCallNode{Function: ast.Ident{ID: ast.Identifier(tt.callee)}}
			err := validateDeferGoBuiltinRestriction("defer", call)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
