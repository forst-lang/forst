package nodert

import (
	"strings"
	"testing"
)

func validEmbeddedManifest() Manifest {
	return Manifest{
		Version: ManifestVersion,
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindFunction},
		},
	}
}

func TestValidateEmbedded_rejectsEmptyExportsAndUnknownKind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		m    Manifest
		want string
	}{
		{
			name: "unsupportedVersion",
			m: Manifest{
				Version: 99,
				Exports: validEmbeddedManifest().Exports,
			},
			want: "unsupported manifest version",
		},
		{
			name: "emptyExports",
			m: Manifest{
				Version: ManifestVersion,
				Exports: nil,
			},
			want: "exports must not be empty",
		},
		{
			name: "unknownKind",
			m: Manifest{
				Version: ManifestVersion,
				Exports: []ExportEntry{
					{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKind("class")},
				},
			},
			want: "unknown export kind",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.m.ValidateEmbedded()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestValidate_requiresBoundaryRoot(t *testing.T) {
	t.Parallel()
	m := validEmbeddedManifest()
	err := m.Validate()
	if err == nil {
		t.Fatal("expected error when boundaryRoot missing")
	}
	if !strings.Contains(err.Error(), "boundaryRoot") {
		t.Fatalf("error = %q", err.Error())
	}

	m.BoundaryRoot = "/tmp/project"
	if err := m.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestAllowCall_rejectsKindMismatch(t *testing.T) {
	t.Parallel()
	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: "/tmp/project",
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindAsyncFunction},
		},
	}
	err := manifest.AllowCall("legacy/payment.ts", "create", ExportKindFunction)
	if err == nil {
		t.Fatal("expected kind mismatch error")
	}
	if !strings.Contains(err.Error(), "expected") || !strings.Contains(err.Error(), string(ExportKindFunction)) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestAllowCall_rejectsKindMismatch_syncExportAsyncCall(t *testing.T) {
	t.Parallel()
	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: "/tmp/project",
		Exports: []ExportEntry{
			{ModuleID: "legacy/payment.ts", Name: "create", Kind: ExportKindFunction},
		},
	}
	err := manifest.AllowCall("legacy/payment.ts", "create", ExportKindAsyncFunction)
	if err == nil {
		t.Fatal("expected kind mismatch error")
	}
	if !strings.Contains(err.Error(), string(ExportKindAsyncFunction)) {
		t.Fatalf("error = %q", err.Error())
	}
}
