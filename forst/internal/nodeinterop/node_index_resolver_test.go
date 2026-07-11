package nodeinterop

import (
	"encoding/json"
	"testing"

	"forst/internal/ast"
)

func TestParseIndexV1_valid(t *testing.T) {
	raw := []byte(`{
		"moduleId": "legacy/payment.ts",
		"exports": [{
			"name": "create",
			"kind": "asyncFunction",
			"parameters": [
				{"name": "amount", "type": {"kind": "number"}},
				{"name": "currency", "type": {"kind": "string"}}
			],
			"returnType": {"kind": "object", "fields": {"id": {"kind": "string"}}}
		}]
	}`)
	idx, err := ParseIndexV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	if idx.ModuleID != "legacy/payment.ts" {
		t.Fatalf("moduleId = %q", idx.ModuleID)
	}
	exp, ok := idx.ExportByName("create")
	if !ok || exp.Kind != ExportKindAsyncFunction {
		t.Fatalf("export: %+v ok=%v", exp, ok)
	}
}

func TestIndexResolver_ExportSignature_asyncFunction(t *testing.T) {
	idx := &IndexV1{
		ModuleID: "legacy/payment.ts",
		Exports: []IndexExport{{
			Name: "create",
			Kind: ExportKindAsyncFunction,
			Parameters: []IndexParam{
				{Name: "amount", Type: IndexType{Kind: "number"}},
				{Name: "currency", Type: IndexType{Kind: "string"}},
			},
			ReturnType: &IndexType{
				Kind:   "object",
				Fields: map[string]IndexType{"id": {Kind: "string"}},
			},
		}},
	}
	r := NewIndexResolver()
	if err := r.Register(idx); err != nil {
		t.Fatal(err)
	}
	params, returns, kind, err := r.ExportSignature("legacy/payment.ts", "create")
	if err != nil {
		t.Fatal(err)
	}
	if kind != ExportKindAsyncFunction {
		t.Fatalf("kind = %q", kind)
	}
	if len(params) != 2 || len(returns) != 1 {
		t.Fatalf("params=%d returns=%d", len(params), len(returns))
	}
	if returns[0].Ident == ast.TypeIdent("Task") {
		t.Fatalf("asyncFunction should map to promise element type, got Task")
	}
	if returns[0].Assertion == nil {
		t.Fatalf("expected object return with assertion, got %+v", returns[0])
	}
}

func TestManifestFromIndexes_buildsAllowlist(t *testing.T) {
	idx := &IndexV1{
		ModuleID: "legacy/payment.ts",
		Exports: []IndexExport{
			{Name: "create", Kind: ExportKindAsyncFunction},
			{Name: "watchEvents", Kind: ExportKindAsyncGenerator, YieldType: &IndexType{Kind: "string"}},
		},
	}
	m, err := ManifestFromIndexes(t.TempDir(), []*IndexV1{idx})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Exports) != 2 {
		t.Fatalf("exports: %+v", m.Exports)
	}
	raw, err := m.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded ManifestV1
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
}
