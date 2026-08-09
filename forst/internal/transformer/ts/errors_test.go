package transformerts

import (
	"fmt"
	"strings"
	"testing"
)

const testNpmPackage = "@forst/gen"

func TestEmitErrorsESM_reExportsInvokeFromSharedPackage(t *testing.T) {
	got := EmitErrorsESM(testNpmPackage, nil, RuntimePromise)
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
	got := EmitErrorsDTS(testNpmPackage, nil, RuntimePromise)
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

func TestEmitErrorsESM_effectModeUsesEffectSubpath(t *testing.T) {
	got := EmitErrorsESM(testNpmPackage, []ErrorClass{{
		Name: "CellTaken",
		Tag:  "CellTaken",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}}, RuntimeEffect)
	assertContainsAll(t, got, []string{
		`import { ForstUnknownFailure } from "@forst/errors/effect"`,
		`extends Data.TaggedError("@forst/gen/CellTaken")`,
		`from "@forst/errors/effect"`,
	})
	assertContainsNone(t, got, []string{"const tagged =", "./invoke-errors.js"})
}

func TestEmitErrorsESM_generatesDomainClassesAndReExports(t *testing.T) {
	cellTaken := ErrorClass{
		Name: "CellTaken",
		Tag:  "CellTaken",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}
	got := EmitErrorsESM(testNpmPackage, []ErrorClass{cellTaken}, RuntimePromise)
	if strings.Count(got, "export {") != 1 {
		t.Fatalf("expected one shared re-export block:\n%s", got)
	}
	assertContainsAll(t, got, []string{
		`import { ForstUnknownFailure } from "@forst/errors"`,
		`extends tagged("@forst/gen/CellTaken")`,
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
		`ForstUnknownFailure`,
		"ForstTestServerFailed",
		"InvokeRejected",
	})
	assertContainsNone(t, got, []string{
		`extends tagged("ForstUnknownFailure")`,
	})
}

func TestEmitErrorsDTS_exportsForstErrorUnion(t *testing.T) {
	cellTaken := ErrorClass{
		Name: "CellTaken",
		Tag:  "CellTaken",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}
	got := EmitErrorsDTS(testNpmPackage, []ErrorClass{cellTaken}, RuntimePromise)
	assertContainsAll(t, got, []string{
		"export type ForstError =",
		"export declare function decodeDomainError",
		`from "@forst/errors"`,
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
		if err := ValidateDomainErrors([]ErrorClass{{Name: "CellTaken", Tag: "CellTaken"}}); err != nil {
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
	got := EmitErrorsESM(testNpmPackage, nil, RuntimePromise)
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
