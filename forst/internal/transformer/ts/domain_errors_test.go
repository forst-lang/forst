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

func TestTransportDecode_usesPackageScopedWireTag(t *testing.T) {
	block := transportDomainErrorDecodeBlock([]PackageDomainErrorEmit{{
		ForstPackage: "main",
		Errors: []ErrorClass{{
			Name:         "CellTaken",
			ForstPackage: "main",
			WireTag:      "main/CellTaken",
		}},
	}}, RuntimePromise)
	if !stringsContains(block, `"main/CellTaken": $main.CellTaken`) {
		t.Fatalf("missing package-scoped registry key:\n%s", block)
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
	want := map[string]string{
		"alpha/NotFound": "alpha",
		"beta/NotFound":  "beta",
	}
	for _, c := range merged {
		key := c.ForstPackage + "/" + c.Name
		if wantPkg, ok := want[key]; !ok {
			t.Fatalf("unexpected merged entry %+v", c)
		} else if c.ForstPackage != wantPkg || c.Name != "NotFound" {
			t.Fatalf("merged entry = %+v, want package %q name NotFound", c, wantPkg)
		}
	}
}

func TestMergeDomainErrors_rejectsMissingForstPackage(t *testing.T) {
	_, err := MergeDomainErrors([]ErrorClass{{Name: "NotFound"}})
	if err == nil {
		t.Fatal("expected error for missing Forst package")
	}
	if !strings.Contains(err.Error(), "NotFound") || !strings.Contains(err.Error(), "missing Forst package") {
		t.Fatalf("unexpected error: %v", err)
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
