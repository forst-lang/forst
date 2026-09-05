package ast

import "testing"

func TestIntLiteralNode_Source_preservesRawAndSign(t *testing.T) {
	t.Parallel()
	cases := []struct {
		n    IntLiteralNode
		want string
	}{
		{IntLiteralNode{Value: 54, Raw: "0x36"}, "0x36"},
		{IntLiteralNode{Value: -16, Raw: "0x10"}, "-0x10"},
		{IntLiteralNode{Value: 42}, "42"},
		{IntLiteralNode{Value: -7}, "-7"},
		{IntLiteralNode{Value: 63, Raw: "0o77"}, "0o77"},
		{IntLiteralNode{Value: 10, Raw: "0b1010"}, "0b1010"},
	}
	for _, tc := range cases {
		if got := tc.n.Source(); got != tc.want {
			t.Fatalf("%#v.Source() = %q, want %q", tc.n, got, tc.want)
		}
	}
}
