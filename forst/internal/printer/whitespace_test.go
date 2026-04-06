package printer

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
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

func TestUTF16Len_nonBMP(t *testing.T) {
	t.Parallel()
	// One Unicode scalar value above U+FFFF uses two UTF-16 code units.
	if got := UTF16Len("\U0001F600"); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestFormatSource_unexpectedTopLevel_returnsError(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	_, err := FormatSource("unexpected\n", "t.ft", log)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestFormatDocument_parseError_fallsBackToWhitespaceOnly(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	src := "unexpected\n"
	out := FormatDocument(src, "bad.ft", 2, true, log)
	want := FormatForstWhitespace(src, 2, true)
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestFormatDocument_validSource_appliesWhitespacePass(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	src := `package main

func main() {
}
`
	out := FormatDocument(src, "ok.ft", 2, true, log)
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("expected final newline: %q", out)
	}
}
