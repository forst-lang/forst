package transformerts

import (
	"fmt"
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
	cellTaken := ErrorClass{
		Name: "CellTaken",
		Tag:  "CellTaken",
		Fields: []ErrorField{
			{Name: "row", TSType: "number"},
			{Name: "col", TSType: "number"},
		},
	}
	got := EmitDomainErrorsDTS(testNpmPackage, []ErrorClass{cellTaken}, RuntimePromise)
	assertContainsAll(t, got, []string{
		"export type ForstError =",
		"export declare function decodeDomainError",
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

func TestEmitHarnessErrorESM_namespacesHarnessTags(t *testing.T) {
	got := EmitHarnessErrorESM(testNpmPackage, RuntimePromise)
	for _, name := range HarnessErrorClassNames() {
		tag := forstBuiltInTag(name)
		if !strings.Contains(got, `extends tagged("`+tag+`")`) {
			t.Fatalf("missing built-in harness tag %q:\n%s", tag, got)
		}
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
