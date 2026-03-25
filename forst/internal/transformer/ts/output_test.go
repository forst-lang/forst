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
