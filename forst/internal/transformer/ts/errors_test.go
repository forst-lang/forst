package transformerts

import (
	"fmt"
	"strings"
	"testing"
)

const testNpmPackage = "@forst/gen"

func mustEmitErrorsESM(t *testing.T, npm string, domain []ErrorClass, runtime ClientRuntime) string {
	t.Helper()
	got, err := EmitErrorsESM(npm, domain, runtime)
	if err != nil {
		t.Fatalf("EmitErrorsESM: %v", err)
	}
	return got
}

func mustEmitErrorsDTS(t *testing.T, npm string, domain []ErrorClass, runtime ClientRuntime) string {
	t.Helper()
	got, err := EmitErrorsDTS(npm, domain, runtime)
	if err != nil {
		t.Fatalf("EmitErrorsDTS: %v", err)
	}
	return got
}

func TestEmitErrorsESM_reExportsInvokeFromSharedPackage(t *testing.T) {
	got := mustEmitErrorsESM(t, testNpmPackage, nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		`from "@forst/errors"`,
		"InvokeRejected",
		"isInvokeFailure",
		"ForstUnknownFailure",
		"ForstTestServerFailed",
	})
	assertContainsNone(t, got, []string{
		"const tagged =",
		`from "./invoke-errors.js"`,
		"export class InvokeRejected",
	})
}

func TestEmitErrorsDTS_reExportsInvokeFailureType(t *testing.T) {
	got := mustEmitErrorsDTS(t, testNpmPackage, nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		`from "@forst/errors"`,
		"export type { InvokeFailure }",
	})
	for _, name := range ErrorClassNames() {
		if !strings.Contains(got, name) {
			t.Fatalf("missing %s in errors re-export:\n%s", name, got)
		}
	}
}

func TestEmitPackageDomainErrorsESM_effectModeUsesPackageScopedTag(t *testing.T) {
	got, err := EmitPackageDomainErrorsESM(testNpmPackage, "auth", []ErrorClass{{
		Name:         "CellTaken",
		ForstPackage: "auth",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}}, RuntimeEffect)
	if err != nil {
		t.Fatal(err)
	}
	assertContainsAll(t, got, []string{
		`extends Data.TaggedError("@forst/gen/auth/CellTaken")`,
	})
	assertContainsNone(t, got, []string{"const tagged =", "./invoke-errors.js"})
}

func TestEmitErrorsESM_generatesRegistryAndReExports(t *testing.T) {
	cellTaken := ErrorClass{
		Name:         "CellTaken",
		ForstPackage: "main",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}
	got := mustEmitErrorsESM(t, testNpmPackage, []ErrorClass{cellTaken}, RuntimePromise)
	if strings.Count(got, "export {") != 2 {
		t.Fatalf("expected domain + shared re-export blocks:\n%s", got)
	}
	assertContainsAll(t, got, []string{
		`import { ForstUnknownFailure } from "@forst/errors"`,
		`import { CellTaken } from "./pkg/main.errors.js"`,
		`"main/CellTaken": CellTaken`,
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
		"export {\n  CellTaken,\n};",
		"ForstTestServerFailed",
		"InvokeRejected",
	})
	assertContainsNone(t, got, []string{
		`extends tagged("ForstUnknownFailure")`,
		"const tagged =",
	})
}

func TestEmitErrorsDTS_exportsForstErrorUnion(t *testing.T) {
	cellTaken := ErrorClass{
		Name:         "CellTaken",
		ForstPackage: "main",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}
	got := mustEmitErrorsDTS(t, testNpmPackage, []ErrorClass{cellTaken}, RuntimePromise)
	assertContainsAll(t, got, []string{
		"export type ForstError =",
		"export declare function decodeDomainError",
		`from "@forst/errors"`,
		`import { CellTaken } from "./pkg/main.errors.js"`,
		`ForstUnknownFailure`,
	})
	unionIdx := strings.Index(got, "export type ForstError =")
	if unionIdx < 0 {
		t.Fatal("missing ForstError type")
	}
	rest := got[unionIdx:]
	end := strings.Index(rest, ";\n\n")
	if end < 0 {
		t.Fatalf("malformed ForstError type:\n%s", rest)
	}
	forstErrorDecl := rest[:end+1]
	for _, frag := range []string{"| CellTaken", "| ForstUnknownFailure"} {
		if !strings.Contains(forstErrorDecl, frag) {
			t.Fatalf("ForstError union missing %q:\n%s", frag, forstErrorDecl)
		}
	}
	assertContainsNone(t, got, []string{
		"export declare class ForstUnknownFailure",
	})
}

func TestValidateDomainErrors_rejectsReservedNames(t *testing.T) {
	for _, name := range ReservedClientErrorNames() {
		t.Run(name, func(t *testing.T) {
			err := ValidateDomainErrors([]ErrorClass{{Name: name, Tag: name}})
			if err == nil {
				t.Fatalf("expected collision error for %q", name)
			}
			want := fmt.Sprintf("domain error %q conflicts with reserved Forst client error name", name)
			if err.Error() != want {
				t.Fatalf("error = %q, want %q", err.Error(), want)
			}
		})
	}
	t.Run("allows non-reserved names", func(t *testing.T) {
		if err := ValidateDomainErrors([]ErrorClass{{Name: "CellTaken", Tag: "CellTaken", ForstPackage: "main"}}); err != nil {
			t.Fatalf("CellTaken should be allowed: %v", err)
		}
	})
}

func TestEmitHarnessErrorESM_reExportsFromSharedPackage(t *testing.T) {
	got := EmitHarnessErrorESM(testNpmPackage, RuntimePromise)
	assertContainsAll(t, got, []string{
		`import { ForstTestServerFailed } from "@forst/errors"`,
		"export { ForstTestServerFailed }",
	})
	assertContainsNone(t, got, []string{"const tagged ="})
}

func TestEmitErrorsESM_emptyDomainSingleReExportBlock(t *testing.T) {
	got := mustEmitErrorsESM(t, testNpmPackage, nil, RuntimePromise)
	if strings.Count(got, "export {") != 1 {
		t.Fatalf("export blocks = %d, want 1:\n%s", strings.Count(got, "export {"), got)
	}
}

func TestEmitHarnessErrorESM_effectModeUsesEffectSubpath(t *testing.T) {
	got := EmitHarnessErrorESM(testNpmPackage, RuntimeEffect)
	assertContainsAll(t, got, []string{
		`import { ForstTestServerFailed } from "@forst/errors/effect"`,
		"export { ForstTestServerFailed }",
	})
	assertContainsNone(t, got, []string{"const tagged ="})
}

func TestEmitHarnessErrorDTS_importsTypeForLocalUse(t *testing.T) {
	got := EmitHarnessErrorDTS(testNpmPackage, RuntimeEffect)
	assertContainsAll(t, got, []string{
		`import type { ForstTestServerFailed } from "@forst/errors/effect"`,
		`export { ForstTestServerFailed } from "@forst/errors/effect"`,
	})
}
