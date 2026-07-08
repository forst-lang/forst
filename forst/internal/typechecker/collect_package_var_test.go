package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestParseForstSiblingTypeRef(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in          string
		importLocal string
		typeName    string
		ok          bool
	}{
		{"auth.Logger", "auth", "Logger", true},
		{"Logger", "", "", false},
		{"a.b.c", "", "", false},
		{".Logger", "", "", false},
		{"auth.", "", "", false},
	}
	for _, tt := range tests {
		importLocal, typeName, ok := parseForstSiblingTypeRef(ast.TypeIdent(tt.in))
		if ok != tt.ok || importLocal != tt.importLocal || typeName != tt.typeName {
			t.Errorf("parseForstSiblingTypeRef(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.in, importLocal, typeName, ok, tt.importLocal, tt.typeName, tt.ok)
		}
	}
}

func TestCollectPackageLevelVar_literalOnlyDuringCollect(t *testing.T) {
	tc := New(nil, false)
	n := ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "Version"}}},
		RValues:        []ast.ExpressionNode{ast.StringLiteralNode{Value: "1.0"}},
	}
	if err := tc.collectPackageLevelVar(n); err != nil {
		t.Fatal(err)
	}
	if _, ok := tc.globalScope().LookupVariableType("Version"); !ok {
		t.Fatal("expected Version registered during collect for literal initializer")
	}
}

func TestEnsurePackageLevelVarRegistered_deferredNonLiteral(t *testing.T) {
	tc := New(nil, false)
	n := ast.AssignmentNode{
		IsPackageLevel: true,
		LValues:        []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: "N"}}},
		RValues:        []ast.ExpressionNode{ast.FunctionCallNode{
			Function:  ast.Ident{ID: "len"},
			Arguments: []ast.ExpressionNode{ast.StringLiteralNode{Value: "x"}},
		}},
	}
	if err := tc.collectPackageLevelVar(n); err != nil {
		t.Fatal(err)
	}
	if _, ok := tc.globalScope().LookupVariableType("N"); ok {
		t.Fatal("non-literal package var should not be registered during collect")
	}
	if err := tc.ensurePackageLevelVarRegistered(n); err != nil {
		t.Fatal(err)
	}
	if _, ok := tc.globalScope().LookupVariableType("N"); !ok {
		t.Fatal("expected N registered after ensurePackageLevelVarRegistered")
	}
}
