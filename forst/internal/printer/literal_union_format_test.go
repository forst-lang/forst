package printer

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFormatSource_literalUnionLayout(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	tests := []struct {
		name string
		src  string
		check func(t *testing.T, out string)
	}{
		{
			name: "two-member-inline",
			src:  "package main\n\ntype Pair = \"a\" | \"b\"\n",
			check: func(t *testing.T, out string) {
				t.Helper()
				if strings.Contains(out, "type Pair =\n") {
					t.Fatalf("2-member union should stay inline, got:\n%s", out)
				}
				if !strings.Contains(out, `"a"`) || !strings.Contains(out, `"b"`) || !strings.Contains(out, "|") {
					t.Fatalf("expected inline 2-member union, got:\n%s", out)
				}
			},
		},
		{
			name: "three-member-multiline",
			src: "package main\n\ntype Status =\n\t| \"todo\"\n\t| \"in_progress\"\n\t| \"done\"\n",
			check: func(t *testing.T, out string) {
				t.Helper()
				if !strings.Contains(out, "type Status =\n") {
					t.Fatalf("3+ member union should be multiline, got:\n%s", out)
				}
				for _, lit := range []string{`"todo"`, `"in_progress"`, `"done"`} {
					if !strings.Contains(out, lit) {
						t.Fatalf("missing %s in:\n%s", lit, out)
					}
				}
				if !strings.Contains(out, "|") {
					t.Fatalf("expected leading | members, got:\n%s", out)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := FormatSource(tc.src, "literal_union.ft", log)
			if err != nil {
				t.Fatalf("FormatSource: %v", err)
			}
			tc.check(t, out)
			out2, err := FormatSource(out, "literal_union.ft", log)
			if err != nil {
				t.Fatalf("idempotent FormatSource: %v", err)
			}
			if out != out2 {
				t.Fatalf("formatter not idempotent:\n---\n%s\n---\n%s", out, out2)
			}
		})
	}
}
