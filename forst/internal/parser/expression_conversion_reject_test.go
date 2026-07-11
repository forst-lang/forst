package parser

import (
	"strings"
	"testing"

	"forst/internal/ast"
)

func TestParseFile_rejectsUppercaseTypeConversions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{
			name:    "Int literal",
			body:    "_ = Int(1)",
			wantSub: "not a conversion",
		},
		{
			name:    "String variable",
			body:    "_ = String(x)",
			wantSub: "not a conversion",
		},
		{
			name:    "Float",
			body:    "_ = Float(n)",
			wantSub: "not a conversion",
		},
		{
			name:    "Bool",
			body:    "_ = Bool(flag)",
			wantSub: "not a conversion",
		},
		{
			name:    "slice Int",
			body:    "_ = []Int(s)",
			wantSub: "not a slice conversion",
		},
		{
			name:    "slice String",
			body:    "_ = []String(s)",
			wantSub: "not a slice conversion",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := "package main\n\nfunc main() {\n\t" + tc.body + "\n}\n"
			err := parseShouldFail(src)
			if err == nil {
				t.Fatalf("expected parse error for %s", tc.body)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q should contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestParseFile_allowsByteSliceConversion(t *testing.T) {
	t.Parallel()
	src := `package main

func f(s String) {
	_ = []byte(s)
}
`
	if _, err := NewTestParser(src, ast.SetupTestLogger(nil)).ParseFile(); err != nil {
		t.Fatal(err)
	}
}
