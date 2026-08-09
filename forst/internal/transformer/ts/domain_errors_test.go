package transformerts

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestDomainErrorClassFromTypeDef(t *testing.T) {
	tc := typechecker.New(nil, false)
	tc.Defs["CellTaken"] = ast.TypeDefNode{
		Ident: "CellTaken",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"row": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
					"col": {Type: &ast.TypeNode{Ident: ast.TypeInt}},
				},
			},
		},
	}
	cls, err := DomainErrorClassFromTypeDef(tc.Defs["CellTaken"].(ast.TypeDefNode), tc)
	if err != nil {
		t.Fatal(err)
	}
	if cls.Name != "CellTaken" || len(cls.Fields) != 2 {
		t.Fatalf("cls = %+v", cls)
	}
}

func TestEmitDomainErrorsESM_includesDomainErrorAndRegistry(t *testing.T) {
	out := EmitDomainErrorsESM(testNpmPackage, []ErrorClass{{
		Name: "CellTaken",
		Tag:  "CellTaken",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}})
	for _, want := range []string{
		"export class CellTaken",
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
		"ForstUnknownFailure",
	} {
		if !stringsContains(out, want) {
			t.Fatalf("missing %q in emit", want)
		}
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
