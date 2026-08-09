package transformerts

import (
	"regexp"
	"strings"
	"testing"
)

const testNpmPackage = "@forst/gen"

func TestEmitInvokeErrorsESM_emitsTaggedErrorClasses(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage)
	assertContainsNone(t, got, []string{"from \"effect\"", "from 'effect'", "require(\"effect\")"})
	prefix := clientTagPrefix(testNpmPackage)
	for _, name := range ErrorClassNames() {
		tag := prefix + "/" + name
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

func TestEmitInvokeErrorsESM_errorClassNamesHaveNoErrorSuffix(t *testing.T) {
	re := regexp.MustCompile(`Error$`)
	for _, name := range ErrorClassNames() {
		if re.MatchString(name) {
			t.Fatalf("class name %q must not end in Error", name)
		}
	}
	got := EmitInvokeErrorsESM(testNpmPackage)
	classRe := regexp.MustCompile(`export class (\w+)`)
	for _, m := range classRe.FindAllStringSubmatch(got, -1) {
		if re.MatchString(m[1]) {
			t.Fatalf("emitted class %q ends in Error", m[1])
		}
	}
}

func TestEmitInvokeErrorsESM_taggedErrorCarriesLiteralTag(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage)
	assignIdx := strings.Index(got, "Object.assign(this, props)")
	tagIdx := strings.Index(got, `Object.defineProperty(this, "_tag"`)
	if assignIdx < 0 || tagIdx < 0 || assignIdx > tagIdx {
		t.Fatalf("props must be assigned before _tag is defined:\n%s", got)
	}
	assertContainsAll(t, got, []string{
		"enumerable: true",
		"writable: false",
		"value: tag",
	})
}

func TestEmitInvokeErrorsESM_taggedErrorPropsCannotOverwriteTag(t *testing.T) {
	got := EmitInvokeErrorsESM(testNpmPackage)
	assignIdx := strings.Index(got, "Object.assign(this, props)")
	defineIdx := strings.Index(got, `Object.defineProperty(this, "_tag"`)
	if assignIdx < 0 || defineIdx < 0 || assignIdx > defineIdx {
		t.Fatal("Object.assign must precede defineProperty(_tag) so props cannot win")
	}
	if !strings.Contains(got, "writable: false") {
		t.Fatal("_tag must be non-writable")
	}
}

func TestEmitInvokeErrorsDTS_emitsInvokeFailureUnionAndGuard(t *testing.T) {
	got := EmitInvokeErrorsDTS(testNpmPackage)
	prefix := clientTagPrefix(testNpmPackage)
	assertContainsAll(t, got, []string{
		"export type InvokeFailure =",
		"export declare function isInvokeFailure(u: unknown): u is InvokeFailure",
	})
	for _, name := range ErrorClassNames() {
		if !strings.Contains(got, "export declare class "+name) {
			t.Fatalf("missing DTS class %s:\n%s", name, got)
		}
		if !strings.Contains(got, "| "+name) {
			t.Fatalf("InvokeFailure missing %s:\n%s", name, got)
		}
		tag := prefix + "/" + name
		if !strings.Contains(got, `readonly _tag: "`+tag+`"`) {
			t.Fatalf("missing namespaced _tag for %s:\n%s", name, got)
		}
	}
	for _, name := range HarnessErrorClassNames() {
		if strings.Contains(got, "export declare class "+name) {
			t.Fatalf("invoke-errors must not include harness class %s:\n%s", name, got)
		}
	}
}

func TestEmitInvokeErrorsDTS_emitsInvokeStreamAborted(t *testing.T) {
	got := EmitInvokeErrorsDTS(testNpmPackage)
	prefix := clientTagPrefix(testNpmPackage)
	assertContainsAll(t, got, []string{
		"export declare class InvokeStreamAborted",
		`readonly _tag: "` + prefix + `/InvokeStreamAborted"`,
		"readonly rowIndex: number",
		"| InvokeStreamAborted",
	})
}

func TestEmitInvokeErrorsDTS_extendsErrorAndKeepsInstanceofContract(t *testing.T) {
	got := EmitInvokeErrorsDTS(testNpmPackage)
	for _, name := range ErrorClassNames() {
		frag := "export declare class " + name + " extends Error"
		if !strings.Contains(got, frag) {
			t.Fatalf("expected %q to extend Error", name)
		}
	}
	esm := EmitInvokeErrorsESM(testNpmPackage)
	assertContainsAll(t, esm, []string{
		"class extends Error",
		"Object.setPrototypeOf(this, new.target.prototype)",
		"this.name = tag",
	})
}

func TestEmitDomainErrorsDTS_forstUnknownFailureDoesNotRedeclareMessage(t *testing.T) {
	got := EmitDomainErrorsDTS(testNpmPackage, nil)
	start := strings.Index(got, "export declare class ForstUnknownFailure")
	if start < 0 {
		t.Fatal("missing ForstUnknownFailure class")
	}
	end := strings.Index(got[start:], "}\n\nexport type ForstError")
	if end < 0 {
		end = strings.Index(got[start:], "}\n\nexport declare const DOMAIN_ERROR_REGISTRY")
	}
	if end < 0 {
		t.Fatal("missing end of ForstUnknownFailure class")
	}
	block := got[start : start+end]
	if strings.Count(block, "readonly message") != 1 {
		t.Fatalf("ForstUnknownFailure must declare message only in constructor, got:\n%s", block)
	}
}

func TestEmitInvokeErrors_catalogDrivesBothEmits(t *testing.T) {
	esm := EmitInvokeErrorsESM(testNpmPackage)
	dts := EmitInvokeErrorsDTS(testNpmPackage)
	for _, c := range ErrorCatalog {
		if !strings.Contains(esm, c.Name) || !strings.Contains(dts, c.Name) {
			t.Fatalf("catalog class %s missing from ESM or DTS", c.Name)
		}
		for _, f := range c.Fields {
			if !strings.Contains(dts, f.Name) {
				t.Fatalf("field %s.%s missing from DTS", c.Name, f.Name)
			}
		}
	}
}

func TestValidateDomainErrors_rejectsReservedNames(t *testing.T) {
	err := ValidateDomainErrors([]ErrorClass{{Name: "InvokeRejected", Tag: "InvokeRejected"}})
	if err == nil {
		t.Fatal("expected collision error for InvokeRejected")
	}
}

func TestEmitHarnessErrorESM_namespacesTestServerFailedTag(t *testing.T) {
	got := EmitHarnessErrorESM(testNpmPackage)
	tag := clientTagPrefix(testNpmPackage) + "/TestServerFailed"
	if !strings.Contains(got, `extends tagged("`+tag+`")`) {
		t.Fatalf("missing namespaced harness tag:\n%s", got)
	}
}
