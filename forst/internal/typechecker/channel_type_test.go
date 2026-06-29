package typechecker

import (
	"testing"

	"forst/internal/ast"
)

func TestChannelElementType_chanInt(t *testing.T) {
	tn := ast.TypeNode{
		Ident:      ast.TypeChannel,
		TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}},
	}
	elem, ok := ChannelElementType(tn)
	if !ok || elem.Ident != ast.TypeInt {
		t.Fatalf("elem = %#v ok=%v", elem, ok)
	}
}

func TestChannelElementType_notChannel(t *testing.T) {
	_, ok := ChannelElementType(ast.TypeNode{Ident: ast.TypeInt})
	if ok {
		t.Fatal("Int is not a channel type")
	}
}

func TestIsChannelReturn_singleChanReturn(t *testing.T) {
	fn := &ast.FunctionNode{
		ReturnTypes: []ast.TypeNode{{
			Ident:      ast.TypeChannel,
			TypeParams: []ast.TypeNode{{Ident: ast.TypeString}},
		}},
	}
	if !IsChannelReturn(fn, nil) {
		t.Fatal("expected channel return")
	}
}

func TestIsChannelReturn_multipleReturns(t *testing.T) {
	fn := &ast.FunctionNode{
		ReturnTypes: []ast.TypeNode{
			{Ident: ast.TypeInt},
			{Ident: ast.TypeChannel, TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}}},
		},
	}
	if IsChannelReturn(fn, nil) {
		t.Fatal("multiple returns should not be channel return")
	}
}

func TestIsChannelReturn_fromTypeCheckerSignature(t *testing.T) {
	tc := New(setupTestLogger(nil), false)
	tc.Functions = map[ast.Identifier]FunctionSignature{
		"ch": {
			Ident: ast.Ident{ID: "ch"},
			ReturnTypes: []ast.TypeNode{{
				Ident:      ast.TypeChannel,
				TypeParams: []ast.TypeNode{{Ident: ast.TypeInt}},
			}},
		},
	}
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "ch"}}
	if !IsChannelReturn(&fn, tc) {
		t.Fatal("expected chan Int return from typechecker signature")
	}
}

func TestIsChannelReturn_nilFunction(t *testing.T) {
	if IsChannelReturn(nil, nil) {
		t.Fatal("nil function should not be channel return")
	}
}
