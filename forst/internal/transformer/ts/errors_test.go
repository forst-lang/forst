package transformerts

import (
	"regexp"
	"strings"
	"testing"
)

const testNpmPackage = "@forst/gen"

func TestEmitInvokeErrorsESM_emitsTaggedErrorClasses(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	assertContainsNone(t, got, []string{"from \"effect\"", "from 'effect'", "require(\"effect\")"})
	for _, name := range ErrorClassNames() {
		tag := forstBuiltInTag(name)
		frag := "export class " + name + " extends tagged(\"" + tag + "\")"
		if !strings.Contains(got, frag) {
			t.Fatalf("missing class emit for %s:\n%s", name, got)
		}
	}
	assertContainsAll(t, got, []string{
		"const tagged = (tag) =>",
		"Object.defineProperty(this, \"_tag\"",
		"enumerable: true",
		"writable: false",
		"Object.assign(this, props)",
		"Object.setPrototypeOf(this, new.target.prototype)",
		"export const isInvokeFailure",
		"INVOKE_FAILURE_TAGS",
	})
}

func TestEmitInvokeErrorsESM_namespacesTagsWithPackagePrefix(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	for _, name := range ErrorClassNames() {
		tag := forstBuiltInTag(name)
		if !strings.Contains(got, `"`+tag+`"`) {
			t.Fatalf("missing built-in tag %q:\n%s", tag, got)
		}
	}
}

func TestEmitInvokeErrorsESM_isInvokeFailureUsesTagSet(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	assertContainsAll(t, got, []string{
		"export const isInvokeFailure",
		"INVOKE_FAILURE_TAGS.has(u._tag)",
	})
}

func TestEmitInvokeErrorsESM_noDomainErrors(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	assertContainsNone(t, got, []string{
		"DOMAIN_ERROR_REGISTRY",
		"decodeDomainError",
		"ForstUnknownFailure",
	})
}

func TestEmitInvokeErrorsDTS_exportsInvokeFailureUnion(t *testing.T) {
	got := EmitInvokeErrorsDTS(testNpmPackage, RuntimePromise)
	assertContainsAll(t, got, []string{
		"export type InvokeFailure =",
		"export declare function isInvokeFailure",
	})
	for _, name := range ErrorClassNames() {
		if !strings.Contains(got, name) {
			t.Fatalf("missing %s in InvokeFailure union:\n%s", name, got)
		}
	}
}

func TestEmitInvokeErrorsDTS_extendsErrorAndKeepsInstanceofContract(t *testing.T) {
	got := EmitInvokeErrorsDTS(testNpmPackage, RuntimePromise)
	for _, name := range ErrorClassNames() {
		tag := forstBuiltInTag(name)
		frag := "export declare class " + name + " extends Error"
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %s:\n%s", frag, got)
		}
		if !strings.Contains(got, "_tag: \""+tag+"\"") {
			t.Fatalf("missing _tag for %s:\n%s", name, got)
		}
	}
}

func TestEmitInvokeErrorsDTS_noDomainErrors(t *testing.T) {
	got := EmitInvokeErrorsDTS(testNpmPackage, RuntimePromise)
	assertContainsNone(t, got, []string{
		"ForstError",
		"decodeDomainError",
		"DOMAIN_ERROR_REGISTRY",
	})
}

func TestEmitInvokeErrorsESM_and_DTS_tagNamesMatch(t *testing.T) {
	esm := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	dts := EmitInvokeErrorsDTS(testNpmPackage, RuntimePromise)
	re := regexp.MustCompile(`extends tagged\("([^"]+)"\)`)
	for _, m := range re.FindAllStringSubmatch(esm, -1) {
		tag := m[1]
		if !strings.Contains(dts, "_tag: \""+tag+"\"") {
			t.Fatalf("DTS missing _tag %q from ESM", tag)
		}
	}
}

func TestEmitDomainErrorsDTS_exportsForstErrorUnion(t *testing.T) {
	got := EmitDomainErrorsDTS(testNpmPackage, nil, RuntimePromise)
	assertContainsAll(t, got, []string{
		"export type ForstError =",
		"ForstUnknownFailure",
		"export declare function decodeDomainError",
	})
	assertContainsNone(t, got, []string{
		"InvokeRejected",
		"InvokeFailure",
	})
}

func TestEmitInvokeErrorsESM_doesNotExportHarnessError(t *testing.T) {
	esm := EmitInvokeErrorsESM(testNpmPackage, RuntimePromise)
	dts := EmitInvokeErrorsDTS(testNpmPackage, RuntimePromise)
	assertContainsNone(t, esm, []string{"ForstTestServerFailed"})
	assertContainsNone(t, dts, []string{"ForstTestServerFailed"})
}

func TestValidateDomainErrors_rejectsReservedNames(t *testing.T) {
	err := ValidateDomainErrors([]ErrorClass{{Name: "InvokeRejected", Tag: "InvokeRejected"}})
	if err == nil {
		t.Fatal("expected collision error for InvokeRejected")
	}
}

func TestEmitHarnessErrorESM_namespacesTestServerFailedTag(t *testing.T) {
	got := EmitHarnessErrorESM(testNpmPackage, RuntimePromise)
	tag := forstBuiltInTag("ForstTestServerFailed")
	if !strings.Contains(got, `extends tagged("`+tag+`")`) {
		t.Fatalf("missing built-in harness tag:\n%s", got)
	}
}

func TestEmitInvokeErrorsESM_effectModeUsesDataTaggedError(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage, RuntimeEffect)
	assertContainsAll(t, got, []string{
		`import { Data } from "effect"`,
		"Data.TaggedError",
		`extends Data.TaggedError("@forst/InvokeRejected")`,
	})
	assertContainsNone(t, got, []string{"const tagged ="})
}

func TestEmitDomainErrorsESM_effectModeUsesDataTaggedError(t *testing.T) {
	got := EmitDomainErrorsESM(testNpmPackage, []ErrorClass{{
		Name: "CellTaken",
		Tag:  "CellTaken",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}}, RuntimeEffect)
	assertContainsAll(t, got, []string{
		`import { Data } from "effect"`,
		`extends Data.TaggedError("@forst/gen/CellTaken")`,
		`"CellTaken": CellTaken`,
		"DOMAIN_ERROR_REGISTRY",
	})
	assertContainsNone(t, got, []string{"const tagged ="})
}

func TestEmitHarnessErrorESM_effectModeUsesDataTaggedError(t *testing.T) {
	got := EmitHarnessErrorESM(testNpmPackage, RuntimeEffect)
	tag := forstBuiltInTag("ForstTestServerFailed")
	assertContainsAll(t, got, []string{
		`import { Data } from "effect"`,
		`extends Data.TaggedError("` + tag + `")`,
	})
	assertContainsNone(t, got, []string{"const tagged ="})
}
