package ast

import (
	"strings"
	"testing"
)

func TestPackageNode_kind_String_IsMainPackage(t *testing.T) {
	main := PackageNode{Ident: Ident{ID: "main"}}
	if main.Kind() != NodeKindPackage || !main.IsMainPackage() || main.GetIdent() != "main" {
		t.Fatal(main)
	}
	if !strings.Contains(main.String(), "main") {
		t.Fatal(main.String())
	}
	lib := PackageNode{Ident: Ident{ID: "lib"}}
	if lib.IsMainPackage() {
		t.Fatal("lib should not be main")
	}
}
