package transformergo

import (
	"strings"
	"testing"

	"forst/internal/testutil"
)

func TestHarness_mustCompileGo_smoke(t *testing.T) {
	src := `package main
func main() {}
`
	code := MustCompileGo(t, src, testutil.CompileOpts{})
	if !strings.Contains(code, "package main") {
		t.Fatalf("expected package main in output, got: %s", code)
	}
}
