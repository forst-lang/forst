package semantic_test

import (
	"testing"

	"forst/internal/forstpkg"
)

func TestParseHandlersFixture(t *testing.T) {
	paths := []string{
		"testdata/layout/app/api/routes.ft",
		"testdata/layout/app/api/handlers.ft",
	}
	for _, p := range paths {
		if _, err := forstpkg.ParseForstFile(nil, mustAbs(t, p)); err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
	}
}
