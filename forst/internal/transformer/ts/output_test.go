package transformerts

import (
	"strings"
	"testing"
)

func TestTypeScriptOutput_GenerateTypesFile_buildsFromSlicesOnly(t *testing.T) {
	o := &TypeScriptOutput{
		Types: []string{"export interface Foo { x: number; }"},
		Functions: []FunctionSignature{
			{Name: "Bar", ReturnType: "string", Parameters: []Parameter{{Name: "n", Type: "number"}}},
		},
	}
	s := o.GenerateTypesFile()
	if o.TypesFile != s {
		t.Fatal("TypesFile field should match return value")
	}
	if !strings.Contains(s, "export interface Foo") || !strings.Contains(s, "export function Bar") {
		t.Fatalf("unexpected types file:\n%s", s)
	}
}

func TestTypeScriptOutput_GenerateMainClient_returnsMainClientField(t *testing.T) {
	o := &TypeScriptOutput{MainClient: "// main client\n"}
	if o.GenerateMainClient() != "// main client\n" {
		t.Fatalf("got %q", o.GenerateMainClient())
	}
}

func TestTypeScriptOutput_AddExportedTypeName_ignoresEmptyString(t *testing.T) {
	o := &TypeScriptOutput{}
	o.AddExportedTypeName("")
	o.AddExportedTypeName("X")
	if len(o.ExportedTypeNames) != 1 || o.ExportedTypeNames[0] != "X" {
		t.Fatalf("got %#v", o.ExportedTypeNames)
	}
}
