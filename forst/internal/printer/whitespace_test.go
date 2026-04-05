package printer

import (
	"testing"
)

func TestUTF16Len_ASCII(t *testing.T) {
	t.Parallel()
	if got := UTF16Len("abc"); got != 3 {
		t.Fatalf("got %d", got)
	}
}

func TestEndPositionExclusive(t *testing.T) {
	t.Parallel()
	cases := []struct {
		src  string
		want LSPCharPos
	}{
		{"", LSPCharPos{Line: 0, Character: 0}},
		{"a", LSPCharPos{Line: 0, Character: 1}},
		{"a\n", LSPCharPos{Line: 1, Character: 0}},
		{"a\nb", LSPCharPos{Line: 1, Character: 1}},
		{"a\r\nb", LSPCharPos{Line: 1, Character: 1}},
	}
	for _, tc := range cases {
		got := EndPositionExclusive(tc.src)
		if got != tc.want {
			t.Errorf("%q: got %+v want %+v", tc.src, got, tc.want)
		}
	}
}

func TestFormatForstWhitespace_TrimsTrailingSpace(t *testing.T) {
	t.Parallel()
	in := "package main  \nfunc x(): String {\n}\n"
	out := FormatForstWhitespace(in, 4, true)
	if out == in {
		t.Fatal("expected change")
	}
	if out != "package main\nfunc x(): String {\n}\n" {
		t.Fatalf("got %q", out)
	}
}

func TestFormatForstWhitespace_ExpandsLeadingTabsWhenInsertSpaces(t *testing.T) {
	t.Parallel()
	in := "\tfoo\n"
	out := FormatForstWhitespace(in, 2, true)
	if out != "  foo\n" {
		t.Fatalf("got %q", out)
	}
}
