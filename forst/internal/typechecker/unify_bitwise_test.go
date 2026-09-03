package typechecker

import (
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestUnifyTypes_bitwiseInt(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	left := ast.IntLiteralNode{Value: 1}
	right := ast.IntLiteralNode{Value: 2}
	for _, op := range []ast.TokenIdent{
		ast.TokenBitwiseAnd, ast.TokenBitwiseOr, ast.TokenXor,
		ast.TokenLShift, ast.TokenRShift, ast.TokenAndNot,
	} {
		ty, err := tc.unifyTypes(left, right, op)
		if err != nil {
			t.Fatalf("op %s: %v", op, err)
		}
		if ty.Ident != ast.TypeInt {
			t.Fatalf("op %s: got %s want Int", op, ty.Ident)
		}
	}
}

func TestUnifyTypes_bitwiseRejectsBool(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	_, err := tc.unifyTypes(
		ast.BoolLiteralNode{Value: true},
		ast.BoolLiteralNode{Value: false},
		ast.TokenXor,
	)
	if err == nil {
		t.Fatal("expected error for bool xor")
	}
}

func TestUnifyTypes_bitwiseByteAndInt(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

func maskPad(k []byte, i Int): []byte {
	k[i] = k[i] ^ 0x36
	return k
}

func main() {
	b := []byte("hi")
	println(len(maskPad(b, 0)))
}
`)
}

func TestUnifyTypes_bitwiseByteAndByte(t *testing.T) {
	t.Parallel()
	typecheckMustOK(t, `package main

func xorInto(a []byte, b []byte, i Int): []byte {
	a[i] = a[i] ^ b[i]
	return a
}

func main() {
	x := []byte{1}
	y := []byte{2}
	println(len(xorInto(x, y, 0)))
}
`)
}
