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
	if !strings.Contains(s, "export interface Foo") {
		t.Fatalf("unexpected types file:\n%s", s)
	}
	if strings.Contains(s, "export function Bar") || strings.Contains(s, "Function signatures") {
		t.Fatalf("types must be shapes only; function signatures live on package modules, not types.d.ts:\n%s", s)
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
	if strings.Contains(withStream, "export function StreamItems(") {
		t.Fatalf("types must not emit function signatures:\n%s", withStream)
	}
	if !strings.Contains(withStream, "export interface StreamingResult") {
		t.Fatalf("streaming types must include StreamingResult:\n%s", withStream)
	}
	if strings.Contains(withStream, "@forst/sidecar") {
		t.Fatalf("types must not import @forst/sidecar:\n%s", withStream)
	}

	empty := formatTypesDeclarationFile(nil, nil)
	if strings.Contains(empty, "Type definitions") || strings.Contains(empty, "Function signatures") {
		t.Fatalf("did not expect section headers for empty inputs:\n%s", empty)
	}
	if strings.Contains(empty, "StreamingResult") {
		t.Fatalf("non-streaming types must not emit StreamingResult:\n%s", empty)
	}
}

func TestGeneratedTypes_containsStreamingResultAndNoSidecarImport(t *testing.T) {
	o := &TypeScriptOutput{
		Functions: []FunctionSignature{
			{
				Name:             "StreamItems",
				Parameters:       []Parameter{{Name: "limit", Type: "number"}},
				ReturnType:       "AsyncIterable<string>",
				StreamingRowType: "string",
			},
		},
	}
	s := o.GenerateTypesFile()
	for _, frag := range []string{
		"export interface StreamingResult",
		"data: any",
		"status: string",
	} {
		if !strings.Contains(s, frag) {
			t.Fatalf("types missing %q:\n%s", frag, s)
		}
	}
	for _, banned := range []string{
		"@forst/sidecar",
		"@forst/client",
		"import(",
		"export function StreamItems",
		"StreamItemsStream",
	} {
		if strings.Contains(s, banned) {
			t.Fatalf("types must not contain %q:\n%s", banned, s)
		}
	}
}

func TestTypeScriptOutput_GeneratePackageModule_reExportsCoreAndTypes(t *testing.T) {
	o := &TypeScriptOutput{
		PackageName:       "bcrypt",
		ExportedTypeNames: []string{"ComparePasswordRequest", "ComparePasswordResponse"},
	}
	got := o.GeneratePackageModule()
	for _, frag := range []string{
		`export * from "../core/bcrypt.js"`,
		"export type { ComparePasswordRequest, ComparePasswordResponse }",
		`from "../types.js"`,
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q:\n%s", frag, got)
		}
	}
}
