package transformerts

import (
	"strings"
	"testing"
)

func TestMergeTypeScriptOutputs_empty(t *testing.T) {
	out, err := MergeTypeScriptOutputs(nil)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || len(out.Types) != 0 || len(out.Functions) != 0 || len(out.ExportedTypeNames) != 0 {
		t.Fatalf("want empty merged output, got %#v", out)
	}
}

func TestMergeTypeScriptOutputs_skipsNilEntries(t *testing.T) {
	a := &TypeScriptOutput{
		PackageName: "main",
		Types:       []string{"export interface Only {}"},
		Functions: []FunctionSignature{
			{Name: "F", ReturnType: "void"},
		},
	}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{nil, a, nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Types) != 1 || len(out.Functions) != 1 {
		t.Fatalf("got types=%d funcs=%d", len(out.Types), len(out.Functions))
	}
}

func TestMergeTypeScriptOutputs_dedupesIdenticalTypeStrings(t *testing.T) {
	a := &TypeScriptOutput{
		PackageName: "main",
		Types:       []string{"export interface A { x: number; }", "export interface A { x: number; }"},
		Functions:   nil,
	}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{a})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Types) != 1 {
		t.Fatalf("want 1 unique type block, got %d: %v", len(out.Types), out.Types)
	}
}

func TestMergeTypeScriptOutputs_mergesFunctionsAcrossFiles(t *testing.T) {
	a := &TypeScriptOutput{
		PackageName: "main",
		Functions: []FunctionSignature{
			{Name: "Foo", Parameters: []Parameter{{Name: "a", Type: "string"}}, ReturnType: "void"},
		},
	}
	b := &TypeScriptOutput{
		PackageName: "main",
		Functions: []FunctionSignature{
			{Name: "Bar", Parameters: nil, ReturnType: "number"},
		},
	}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Functions) != 2 {
		t.Fatalf("want 2 functions, got %d", len(out.Functions))
	}
}

func TestMergeTypeScriptOutputs_duplicateFunctionSameSignature_ok(t *testing.T) {
	sig := FunctionSignature{Name: "Echo", Parameters: []Parameter{{Name: "x", Type: "string"}}, ReturnType: "string"}
	a := &TypeScriptOutput{Functions: []FunctionSignature{sig}}
	b := &TypeScriptOutput{Functions: []FunctionSignature{sig}}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Functions) != 1 {
		t.Fatalf("want deduped single Echo, got %d", len(out.Functions))
	}
}

func TestMergeTypeScriptOutputs_duplicateFunctionConflictingSignature_errors(t *testing.T) {
	a := &TypeScriptOutput{Functions: []FunctionSignature{{Name: "Echo", ReturnType: "string"}}}
	b := &TypeScriptOutput{Functions: []FunctionSignature{{Name: "Echo", ReturnType: "number"}}}
	_, err := MergeTypeScriptOutputs([]*TypeScriptOutput{a, b})
	if err == nil {
		t.Fatal("expected error for conflicting Echo signatures")
	}
	if !strings.Contains(err.Error(), "Echo") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMergeTypeScriptOutputs_mergesExportedTypeNames(t *testing.T) {
	a := &TypeScriptOutput{ExportedTypeNames: []string{"A", "B"}}
	b := &TypeScriptOutput{ExportedTypeNames: []string{"B", "C"}}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ExportedTypeNames) != 3 {
		t.Fatalf("want A B C deduped, got %v", out.ExportedTypeNames)
	}
}

func TestMergeTypeScriptOutputs_skipsEmptyExportedTypeNameEntries(t *testing.T) {
	a := &TypeScriptOutput{
		PackageName:       "main",
		ExportedTypeNames: []string{"", "A"},
		Types:             []string{"export interface A {}"},
	}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{a})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.ExportedTypeNames) != 1 || out.ExportedTypeNames[0] != "A" {
		t.Fatalf("got %v", out.ExportedTypeNames)
	}
}

func TestMergeTypeScriptOutputs_packageNameFromFirstNonNilWithPackage(t *testing.T) {
	first := &TypeScriptOutput{PackageName: ""}
	second := &TypeScriptOutput{
		PackageName: "pkg",
		Types:       []string{"export interface X { x: number; }"},
	}
	out, err := MergeTypeScriptOutputs([]*TypeScriptOutput{nil, first, second})
	if err != nil {
		t.Fatal(err)
	}
	if out.PackageName != "pkg" {
		t.Fatalf("PackageName: %q", out.PackageName)
	}
}

func TestFunctionSignaturesEqual(t *testing.T) {
	a := FunctionSignature{Name: "f", Parameters: []Parameter{{Name: "a", Type: "string"}}, ReturnType: "void"}
	b := FunctionSignature{Name: "f", Parameters: []Parameter{{Name: "a", Type: "string"}}, ReturnType: "void"}
	if !functionSignaturesEqual(a, b) {
		t.Fatal("expected equal")
	}
	c := FunctionSignature{Name: "f", Parameters: []Parameter{{Name: "a", Type: "number"}}, ReturnType: "void"}
	if functionSignaturesEqual(a, c) {
		t.Fatal("expected not equal")
	}
}

func TestFunctionSignaturesEqual_falseWhenNameOrReturnOrArityDiffers(t *testing.T) {
	base := FunctionSignature{Name: "f", Parameters: []Parameter{{Name: "a", Type: "string"}}, ReturnType: "void"}
	if functionSignaturesEqual(base, FunctionSignature{Name: "g", Parameters: base.Parameters, ReturnType: "void"}) {
		t.Fatal("name")
	}
	if functionSignaturesEqual(base, FunctionSignature{Name: "f", Parameters: base.Parameters, ReturnType: "number"}) {
		t.Fatal("return")
	}
	if functionSignaturesEqual(base, FunctionSignature{Name: "f", Parameters: []Parameter{{Name: "a", Type: "string"}, {Name: "b", Type: "string"}}, ReturnType: "void"}) {
		t.Fatal("param count")
	}
	otherParamName := FunctionSignature{Name: "f", Parameters: []Parameter{{Name: "b", Type: "string"}}, ReturnType: "void"}
	if functionSignaturesEqual(base, otherParamName) {
		t.Fatal("param name")
	}
}
