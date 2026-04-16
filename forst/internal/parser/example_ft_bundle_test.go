package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestParseFile_examplesBundle parses large example files to cover parser edge paths.
func TestParseFile_examplesBundle(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	rel := []string{
		"basic.ft",
		"loop.ft",
		"generics.ft",
		"go_builtins.ft",
		"ensure.ft",
		"pointers.ft",
		"union_error_types.ft",
		"union_error_narrowing.ft",
		"nominal_error.ft",
		"rfc/guard/shape_guard.ft",
		"rfc/guard/basic_guard.ft",
	}
	root := filepath.Join("..", "..", "..", "examples", "in")
	for _, name := range rel {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(root, name)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			pr := NewTestParser(string(src), log)
			if _, err := pr.ParseFile(); err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
		})
	}
}
