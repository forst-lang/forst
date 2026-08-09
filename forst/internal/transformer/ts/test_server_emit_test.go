package transformerts

import (
	"strings"
	"testing"
)

func TestEmitHarnessError_reExportsHarnessOnly(t *testing.T) {
	esm := EmitHarnessErrorESM(testNpmPackage, RuntimePromise)
	dts := EmitHarnessErrorDTS(testNpmPackage, RuntimePromise)
	assertContainsAll(t, esm, []string{
		`import { ForstTestServerFailed } from "@forst/errors"`,
		"export { ForstTestServerFailed }",
	})
	assertContainsAll(t, dts, []string{
		`import type { ForstTestServerFailed } from "@forst/errors"`,
		`export { ForstTestServerFailed } from "@forst/errors"`,
	})
	if strings.Contains(esm, "InvokeRejected") {
		t.Fatal("harness module must not mention InvokeRejected")
	}
}
