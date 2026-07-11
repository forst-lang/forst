package nodeinterop

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleManifest() *ManifestV1 {
	return &ManifestV1{
		Version:      ManifestVersionV1,
		BoundaryRoot: "/abs/path/to/project",
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "watchEvents", Kind: ExportKindAsyncGenerator},
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindAsyncFunction},
		},
	}
}

func TestManifestV1_Validate_valid(t *testing.T) {
	if err := sampleManifest().Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestManifestV1_Validate_rejectsBadVersion(t *testing.T) {
	m := sampleManifest()
	m.Version = 2
	if err := m.Validate(); err == nil {
		t.Fatal("expected version error")
	}
}

func TestManifestV1_Validate_rejectsRelativeBoundaryRoot(t *testing.T) {
	m := sampleManifest()
	m.BoundaryRoot = "relative/path"
	if err := m.Validate(); err == nil {
		t.Fatal("expected boundaryRoot error")
	}
}

func TestManifestV1_Validate_rejectsInvalidExportKind(t *testing.T) {
	m := sampleManifest()
	m.Exports[0].Kind = "promise"
	if err := m.Validate(); err == nil {
		t.Fatal("expected kind error")
	}
}

func TestManifestV1_Validate_rejectsDuplicateExport(t *testing.T) {
	m := sampleManifest()
	m.Exports = append(m.Exports, m.Exports[1])
	if err := m.Validate(); err == nil {
		t.Fatal("expected duplicate export error")
	}
}

func TestManifestV1_Validate_rejectsBadModuleID(t *testing.T) {
	cases := []string{
		"../escape.ts",
		"/abs/payment.ts",
		"file://legacy/payment.ts",
		"legacy/payment.go",
		"",
	}
	for _, moduleID := range cases {
		t.Run(moduleID, func(t *testing.T) {
			m := sampleManifest()
			m.Exports[0].ModuleID = moduleID
			if err := m.Validate(); err == nil {
				t.Fatalf("expected error for moduleId %q", moduleID)
			}
		})
	}
}

func TestManifestV1_EmbeddedManifestJSON_omitsBoundaryRoot(t *testing.T) {
	m := sampleManifest()
	raw, err := m.EmbeddedManifestJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "boundaryRoot") {
		t.Fatalf("embedded manifest must not contain boundaryRoot: %s", raw)
	}
	var decoded EmbeddedManifestV1
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != ManifestVersionV1 {
		t.Fatalf("version = %d", decoded.Version)
	}
	if len(decoded.Exports) != 2 {
		t.Fatalf("exports: %+v", decoded.Exports)
	}
}

func TestManifestV1_CanonicalJSON_sortsExports(t *testing.T) {
	m := sampleManifest()
	got, err := m.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded ManifestV1
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Exports) != 2 {
		t.Fatalf("exports: %+v", decoded.Exports)
	}
	if decoded.Exports[0].Name != "create" || decoded.Exports[1].Name != "watchEvents" {
		t.Fatalf("expected sorted exports, got %+v", decoded.Exports)
	}
}

func TestManifestV1_roundTrip(t *testing.T) {
	m := sampleManifest()
	raw, err := m.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseManifestV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := parsed.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(again) {
		t.Fatalf("round-trip canonical mismatch:\nfirst:  %s\nsecond: %s", raw, again)
	}
}

func TestParseManifestV1_invalidJSON(t *testing.T) {
	_, err := ParseManifestV1([]byte("{not json"))
	if err == nil {
		t.Fatal("expected parse error")
	}
}
