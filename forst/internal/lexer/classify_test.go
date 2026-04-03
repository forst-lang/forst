package lexer

import "testing"

func TestIsSpecialChar(t *testing.T) {
	special := []byte{'(', ')', '{', '}', ':', ',', '+', '-', '*', '/', '%', '=', '!', '>', '<', '&', '|', '.', '[', ']', ';'}
	for _, c := range special {
		if !isSpecialChar(c) {
			t.Errorf("isSpecialChar(%q) = false", c)
		}
	}
	if isSpecialChar('a') || isSpecialChar(' ') {
		t.Fatal("expected non-special for letter and space")
	}
}

func TestIsTwoCharOperator(t *testing.T) {
	yes := []string{"->", "==", "!=", ">=", "<=", "&&", "||", ":=", "//", "++", "--"}
	for _, s := range yes {
		if !isTwoCharOperator(s) {
			t.Errorf("isTwoCharOperator(%q) = false", s)
		}
	}
	if isTwoCharOperator("=") || isTwoCharOperator("=>") {
		t.Fatal("single or unknown operator should be false")
	}
}

func TestIsDigit(t *testing.T) {
	for _, c := range []byte{'0', '5', '9'} {
		if !isDigit(c) {
			t.Errorf("isDigit(%q)", c)
		}
	}
	if isDigit('a') {
		t.Fatal("letter should not be digit")
	}
}
