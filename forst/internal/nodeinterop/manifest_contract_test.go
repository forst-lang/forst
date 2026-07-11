package nodeinterop

import (
	"encoding/json"
	"testing"
)

// Normative fixture from .plans/node-interop/SPEC.md (forst-node-manifest-v1).
const specManifestFixture = `{
  "version": 1,
  "boundaryRoot": "/abs/path/to/project",
  "exports": [
    { "moduleId": "legacy/payment.ts", "name": "create", "kind": "asyncFunction" },
    { "moduleId": "legacy/payment.ts", "name": "watchEvents", "kind": "asyncGenerator" }
  ]
}`

func TestManifestContract_specFixtureRoundTrip(t *testing.T) {
	parsed, err := ParseManifestV1([]byte(specManifestFixture))
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := parsed.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := ParseManifestV1(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Version != ManifestVersionV1 {
		t.Fatalf("version: %d", roundTrip.Version)
	}
	if len(roundTrip.Exports) != 2 {
		t.Fatalf("exports: %+v", roundTrip.Exports)
	}
}

func TestManifestContract_specFixtureExportKinds(t *testing.T) {
	parsed, err := ParseManifestV1([]byte(specManifestFixture))
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{
		"create":      ExportKindAsyncFunction,
		"watchEvents": ExportKindAsyncGenerator,
	}
	for _, exp := range parsed.Exports {
		want, ok := kinds[exp.Name]
		if !ok {
			t.Fatalf("unexpected export %q", exp.Name)
		}
		if exp.Kind != want {
			t.Fatalf("export %q kind: got %q want %q", exp.Name, exp.Kind, want)
		}
		if exp.ModuleID != "legacy/payment.ts" {
			t.Fatalf("export %q moduleId: %q", exp.Name, exp.ModuleID)
		}
	}
}

func TestManifestContract_canonicalJSONStableAcrossReorder(t *testing.T) {
	first, err := ParseManifestV1([]byte(specManifestFixture))
	if err != nil {
		t.Fatal(err)
	}
	reordered := &ManifestV1{
		Version:      first.Version,
		BoundaryRoot: first.BoundaryRoot,
		Exports: []ExportEntry{
			first.Exports[1],
			first.Exports[0],
		},
	}
	a, err := first.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	b, err := reordered.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("canonical mismatch after reorder:\n%s\n%s", a, b)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(a, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"version", "boundaryRoot", "exports"} {
		if _, ok := obj[key]; !ok {
			t.Fatalf("missing key %q in canonical JSON", key)
		}
	}
}
