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

func TestFormatFunctionErrorUnion_compactsInvokeCatalog(t *testing.T) {
	domain := map[string]ErrorClass{
		"EmptyMessage": {Name: "EmptyMessage", Tag: "EmptyMessage"},
	}
	sig := typechecker.FunctionSignature{
		ErrorSet: typechecker.FunctionErrorSet{
			NominalErrors: []ast.TypeIdent{"EmptyMessage"},
		},
	}
	got := FormatFunctionErrorUnion(sig, domain)
	want := "EmptyMessage | InvokeFailure"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	for _, name := range []string{"InvokeRejected", "InvokeHttpFailure", "ContractVersionMismatch"} {
		if stringsContains(got, name) {
			t.Fatalf("union must not expand invoke catalog member %q: %q", name, got)
		}
	}
}

func TestFormatFunctionErrorUnion_invokeOnly(t *testing.T) {
	got := FormatFunctionErrorUnion(typechecker.FunctionSignature{}, nil)
	if got != "InvokeFailure" {
		t.Fatalf("got %q, want InvokeFailure", got)
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
	}}, RuntimePromise)
	for _, want := range []string{
		"export class CellTaken",
		`extends tagged("@forst/gen/CellTaken")`,
		`"CellTaken": CellTaken`,
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
