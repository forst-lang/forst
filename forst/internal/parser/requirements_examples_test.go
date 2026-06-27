package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile_requirementsExamplesUseWith(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "..", "examples", "in", "rfc", "requirements", "02-examples.ft")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	full := string(src)

	// Full 02-examples.ft includes table-driven `[] { ... }` slice literals (line ~158) not yet
	// supported by the parser; verify all use/with-bearing functions parse in isolation.
	snippets := []struct {
		name string
		mark string
	}{
		{"expireToken", "func expireToken("},
		{"getUser", "func getUser("},
		{"TestExpireTokenRejectsExpired", "func TestExpireTokenRejectsExpired("},
		{"TestGetUserReturnsUserFromFakeRepo", "func TestGetUserReturnsUserFromFakeRepo("},
		{"TestExpireTokenWithSharedFixture", "func TestExpireTokenWithSharedFixture("},
	}
	for _, sn := range snippets {
		t.Run(sn.name, func(t *testing.T) {
			body := extractFunctionSnippet(full, sn.mark)
			if body == "" {
				t.Fatalf("could not extract %q from 02-examples.ft", sn.mark)
			}
			wrapped := "package requirements_demo\nimport \"testing\"\n\n" + body
			p := NewTestParser(wrapped, nil)
			if _, err := p.ParseFile(); err != nil {
				t.Fatalf("parse %s: %v", sn.name, err)
			}
		})
	}
}

func extractFunctionSnippet(src, startMark string) string {
	idx := strings.Index(src, startMark)
	if idx < 0 {
		return ""
	}
	rest := src[idx:]
	depth := 0
	started := false
	for i, r := range rest {
		if r == '{' {
			depth++
			started = true
		} else if r == '}' {
			depth--
			if started && depth == 0 {
				return rest[:i+1]
			}
		}
	}
	return ""
}
