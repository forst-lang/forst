package programbuild

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOutputPath_rejectsGoFile(t *testing.T) {
	err := ValidateOutputPath("out/main.go")
	if err == nil || !strings.Contains(err.Error(), "generate --go-out") {
		t.Fatalf("ValidateOutputPath() = %v", err)
	}
}

func TestValidateOutputPath_requiresDir(t *testing.T) {
	err := ValidateOutputPath("")
	if err == nil || !strings.Contains(err.Error(), "requires -o") {
		t.Fatalf("ValidateOutputPath() = %v", err)
	}
}

func TestBinaryFileName_fromEntryStem(t *testing.T) {
	name, err := BinaryFileName("/app/main.ft", "linux")
	if err != nil {
		t.Fatal(err)
	}
	if name != "main" {
		t.Fatalf("name = %q want main", name)
	}
	name, err = BinaryFileName("/app/cmd/api.ft", "windows")
	if err != nil {
		t.Fatal(err)
	}
	if name != "api.exe" {
		t.Fatalf("name = %q want api.exe", name)
	}
}

func TestProgramManifest_Validate(t *testing.T) {
	valid := ProgramManifest{
		SchemaVersion: SchemaVersion,
		Kind:          KindProgram,
		Binary:        "bin/main",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	cases := []ProgramManifest{
		{SchemaVersion: 0, Kind: KindProgram, Binary: "bin/main"},
		{SchemaVersion: SchemaVersion, Kind: "invoke", Binary: "bin/main"},
		{SchemaVersion: SchemaVersion, Kind: KindProgram, Binary: ""},
	}
	for i, m := range cases {
		if err := m.Validate(); err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
}

func TestWriteLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	want := ProgramManifest{
		SchemaVersion:   SchemaVersion,
		Kind:            KindProgram,
		CompilerVersion: "test",
		ContractVersion: ContractVersion,
		Entry:           "main.ft",
		BoundaryRoot:    dir,
		GOOS:            "linux",
		GOARCH:          "amd64",
		EmbeddedInvoke:  true,
		Binary:          "bin/main",
		BuiltAt:         "2026-01-01T00:00:00Z",
	}
	if err := Write(dir, want); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Kind != want.Kind || got.Binary != want.Binary || got.SchemaVersion != want.SchemaVersion {
		t.Fatalf("got = %+v want = %+v", got, want)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	var decoded ProgramManifest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != KindProgram {
		t.Fatalf("json kind = %q", decoded.Kind)
	}
}
