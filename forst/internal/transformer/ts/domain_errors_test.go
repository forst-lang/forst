package transformerts

import (
	"strings"
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
		"EmptyMessage": {Name: "EmptyMessage", Tag: "EmptyMessage", ForstPackage: "main"},
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

func TestEmitPackageDomainErrorsESM_includesDomainErrorClass(t *testing.T) {
	out, err := EmitPackageDomainErrorsESM(testNpmPackage, "main", []ErrorClass{{
		Name:         "CellTaken",
		ForstPackage: "main",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}}, RuntimePromise)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"export class CellTaken",
		`extends tagged("@forst/gen/main/CellTaken")`,
	} {
		if !stringsContains(out, want) {
			t.Fatalf("missing %q in emit", want)
		}
	}
}

func TestEmitErrorsESM_registryUsesPackageScopedWireTag(t *testing.T) {
	got, err := EmitErrorsESM(testNpmPackage, []ErrorClass{{
		Name:         "CellTaken",
		ForstPackage: "main",
	}}, RuntimePromise)
	if err != nil {
		t.Fatal(err)
	}
	if !stringsContains(got, `"main/CellTaken": CellTaken`) {
		t.Fatalf("missing package-scoped registry key:\n%s", got)
	}
}

func TestMergeDomainErrors_keepsSameNameAcrossPackages(t *testing.T) {
	merged, err := MergeDomainErrors(
		[]ErrorClass{{Name: "NotFound", ForstPackage: "alpha"}},
		[]ErrorClass{{Name: "NotFound", ForstPackage: "beta"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("got %d errors, want 2: %+v", len(merged), merged)
	}
}

func TestMergeDomainErrors_conflictingFieldsFail(t *testing.T) {
	_, err := MergeDomainErrors(
		[]ErrorClass{{Name: "NotFound", ForstPackage: "auth", Fields: []ErrorField{{Name: "message", TSType: "string"}}}},
		[]ErrorClass{{Name: "NotFound", ForstPackage: "auth", Fields: []ErrorField{{Name: "code", TSType: "number"}}}},
	)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "auth") || !strings.Contains(err.Error(), "NotFound") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootReexportedDomainErrorNames_skipsCollidingBareNames(t *testing.T) {
	names := RootReexportedDomainErrorNames([]ErrorClass{
		{Name: "NotFound", ForstPackage: "alpha"},
		{Name: "NotFound", ForstPackage: "beta"},
		{Name: "Unique", ForstPackage: "alpha"},
	})
	for _, name := range names {
		if name == "NotFound" {
			t.Fatalf("colliding bare name must not be root re-exported: %v", names)
		}
	}
	foundUnique := false
	for _, name := range names {
		if name == "Unique" {
			foundUnique = true
		}
	}
	if !foundUnique {
		t.Fatalf("expected Unique in root re-exports: %v", names)
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
