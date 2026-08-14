package transformerts

import (
	"fmt"
	"strings"
	"testing"
)

const testNpmPackage = "@forst/gen"

func mustEmitErrorsESM(t *testing.T, domain []PackageDomainErrorEmit, runtime ClientRuntime) string {
	t.Helper()
	got, err := EmitErrorsESM(testNpmPackage, domain, runtime)
	if err != nil {
		t.Fatalf("EmitErrorsESM: %v", err)
	}
	return got
}

func mustEmitErrorsDTS(t *testing.T, domain []PackageDomainErrorEmit, runtime ClientRuntime) string {
	t.Helper()
	got, err := EmitErrorsDTS(testNpmPackage, domain, runtime)
	if err != nil {
		t.Fatalf("EmitErrorsDTS: %v", err)
	}
	return got
}

func TestEmitErrorsESM_domainOnlyStubWhenEmpty(t *testing.T) {
	got := mustEmitErrorsESM(t, nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		`Invoke/harness failures: use @forst/errors directly`,
		"export {};",
	})
	assertContainsNone(t, got, []string{
		"export {\n",
		"InvokeRejected",
		"isInvokeFailure",
		"ForstUnknownFailure",
		`from "@forst/errors";`,
		`from "./pkg/`,
	})
}

func TestEmitErrorsDTS_domainOnlyStubWhenEmpty(t *testing.T) {
	got := mustEmitErrorsDTS(t, nil, RuntimeEffect)
	assertContainsAll(t, got, []string{
		`Invoke/harness failures: use @forst/errors/effect directly`,
		"export {};",
	})
	assertContainsNone(t, got, []string{
		"export type { InvokeFailure }",
		"InvokeRejected",
		`from "@forst/errors/effect";`,
	})
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
	assertContainsNone(t, got, []string{
		"const tagged =",
		"./invoke-errors.js",
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
	})
}

func TestEmitPackageDomainErrorsESM_classesOnly(t *testing.T) {
	got, err := EmitPackageDomainErrorsESM(testNpmPackage, "main", []ErrorClass{{
		Name:         "CellTaken",
		ForstPackage: "main",
		Fields:       []ErrorField{{Name: "row", TSType: "number"}},
	}}, RuntimePromise)
	if err != nil {
		t.Fatal(err)
	}
	assertContainsAll(t, got, []string{
		"export class $CellTaken",
		`extends tagged("@forst/gen/main/CellTaken")`,
	})
	assertContainsNone(t, got, []string{
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
	})
}

func TestEmitErrorsESM_includesDomainPackageNamespaces(t *testing.T) {
	got := mustEmitErrorsESM(t, []PackageDomainErrorEmit{{
		ForstPackage: "main",
		Errors:       []ErrorClass{{Name: "CellTaken", ForstPackage: "main"}},
	}}, RuntimePromise)
	assertContainsAll(t, got, []string{
		`export * as main from "./pkg/main.errors.js"`,
	})
}

func TestEmitPackageDomainErrorsESM_rejectsMissingForstPackage(t *testing.T) {
	_, err := EmitPackageDomainErrorsESM(testNpmPackage, "main", []ErrorClass{{
		Name: "CellTaken",
	}}, RuntimePromise)
	if err == nil {
		t.Fatal("expected validation error")
	}
	want := `domain error "CellTaken" missing Forst package name`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestEmitPackageDomainErrorsDTS_rejectsMissingForstPackage(t *testing.T) {
	_, err := EmitPackageDomainErrorsDTS(testNpmPackage, "main", []ErrorClass{{
		Name: "CellTaken",
	}}, RuntimePromise)
	if err == nil {
		t.Fatal("expected validation error")
	}
	want := `domain error "CellTaken" missing Forst package name`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestEmitErrorsESM_noDomainNamespacesWhenEmpty(t *testing.T) {
	got := mustEmitErrorsESM(t, nil, RuntimePromise)
	if !strings.Contains(got, "export {};") {
		t.Fatalf("expected empty export stub:\n%s", got)
	}
	assertContainsNone(t, got, []string{
		"CellTaken",
		"DOMAIN_ERROR_REGISTRY",
		`from "./pkg/`,
		"export * as",
	})
}

func TestEmitTransportDomainErrorDecode_includesPackageRegistry(t *testing.T) {
	block := transportDomainErrorDecodeBlock([]PackageDomainErrorEmit{{
		ForstPackage: "main",
		Errors: []ErrorClass{{
			Name:         "CellTaken",
			ForstPackage: "main",
			WireTag:      "main/CellTaken",
		}},
	}}, RuntimePromise)
	assertContainsAll(t, block, []string{
		`import * as $main from "../pkg/main.errors.js"`,
		`"main/CellTaken": $main.$CellTaken`,
		"packageDomainErrorRegistries",
		"decodeDomainError",
		"UnknownFailureCtor",
		"export function decodeDomainError",
	})
}

func TestEmitTransportErrorsESM_httpPackageUsesDollarPrefix(t *testing.T) {
	got := EmitTransportErrorsESM([]PackageDomainErrorEmit{{
		ForstPackage: "http",
		Errors: []ErrorClass{{
			Name:         "ServiceUnavailable",
			ForstPackage: "http",
			WireTag:      "http/ServiceUnavailable",
		}},
	}}, RuntimePromise)
	assertContainsAll(t, got, []string{
		`import * as $http from "../pkg/http.errors.js"`,
		`"http/ServiceUnavailable": $http.$ServiceUnavailable`,
	})
	assertContainsNone(t, got, []string{
		`import * as http from`,
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

func TestEmitErrorsESM_emptyDomainUsesExportStub(t *testing.T) {
	got := mustEmitErrorsESM(t, nil, RuntimePromise)
	if !strings.Contains(got, "export {};") {
		t.Fatalf("expected export stub:\n%s", got)
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
