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

func TestFormatTypesDeclarationFile_streamAndEmptySections(t *testing.T) {
	withStream := formatTypesDeclarationFile(nil, []FunctionSignature{
		{
			Name:             "StreamItems",
			Parameters:       []Parameter{{Name: "limit", Type: "number"}},
			ReturnType:       "AsyncIterable<string>",
			StreamingRowType: "string",
		},
	})
	if !strings.Contains(withStream, "export function StreamItems(") {
		t.Fatalf("missing function signature:\n%s", withStream)
	}
	if !strings.Contains(withStream, "export function StreamItemsStream(") {
		t.Fatalf("missing stream declaration:\n%s", withStream)
	}

	empty := formatTypesDeclarationFile(nil, nil)
	if strings.Contains(empty, "Type definitions") || strings.Contains(empty, "Function signatures") {
		t.Fatalf("did not expect section headers for empty inputs:\n%s", empty)
	}
}
