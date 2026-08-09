package transformerts

import (
	"regexp"
	"strings"
	"testing"
)

func TestEmitErrorsESM_emitsTaggedErrorClasses(t *testing.T) {
	got := EmitErrorsESM(nil)
	assertContainsNone(t, got, []string{"from \"effect\"", "from 'effect'", "require(\"effect\")"})
	for _, name := range ErrorClassNames() {
		frag := "export class " + name + " extends tagged(\"" + name + "\")"
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

func TestEmitErrorsESM_errorClassNamesHaveNoErrorSuffix(t *testing.T) {
	re := regexp.MustCompile(`Error$`)
	for _, name := range AllExportedErrorClassNames() {
		if re.MatchString(name) {
			t.Fatalf("class name %q must not end in Error", name)
		}
	}
	got := EmitErrorsESM(nil)
	classRe := regexp.MustCompile(`export class (\w+)`)
	for _, m := range classRe.FindAllStringSubmatch(got, -1) {
		if re.MatchString(m[1]) {
			t.Fatalf("emitted class %q ends in Error", m[1])
		}
	}
}

func TestEmitErrorsESM_taggedErrorCarriesLiteralTag(t *testing.T) {
	got := EmitErrorsESM(nil)
	// Props assigned before _tag defineProperty so caller _tag cannot overwrite.
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

func TestEmitErrorsESM_taggedErrorPropsCannotOverwriteTag(t *testing.T) {
	got := EmitErrorsESM(nil)
	assignIdx := strings.Index(got, "Object.assign(this, props)")
	defineIdx := strings.Index(got, `Object.defineProperty(this, "_tag"`)
	if assignIdx < 0 || defineIdx < 0 || assignIdx > defineIdx {
		t.Fatal("Object.assign must precede defineProperty(_tag) so props cannot win")
	}
	if !strings.Contains(got, "writable: false") {
		t.Fatal("_tag must be non-writable")
	}
}

func TestEmitErrorsDTS_emitsInvokeFailureUnionAndGuard(t *testing.T) {
	got := EmitErrorsDTS(nil)
	assertContainsAll(t, got, []string{
		"export type TaggedError<",
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
		if !strings.Contains(got, `readonly _tag: "`+name+`"`) {
			t.Fatalf("missing literal _tag for %s:\n%s", name, got)
		}
	}
	for _, name := range HarnessErrorClassNames() {
		if !strings.Contains(got, "export declare class "+name) {
			t.Fatalf("missing harness DTS class %s:\n%s", name, got)
		}
		if strings.Contains(got, "| "+name+"\n") || strings.Contains(got, "| "+name+";") {
			t.Fatalf("InvokeFailure must not include harness class %s:\n%s", name, got)
		}
	}
}

func TestEmitErrorsDTS_emitsInvokeStreamAborted(t *testing.T) {
	got := EmitErrorsDTS(nil)
	assertContainsAll(t, got, []string{
		"export declare class InvokeStreamAborted",
		`readonly _tag: "InvokeStreamAborted"`,
		"readonly rowIndex: number",
		"| InvokeStreamAborted",
	})
}

func TestEmitErrorsDTS_extendsErrorAndKeepsInstanceofContract(t *testing.T) {
	got := EmitErrorsDTS(nil)
	for _, name := range ErrorClassNames() {
		frag := "export declare class " + name + " extends Error"
		if !strings.Contains(got, frag) {
			t.Fatalf("expected %q to extend Error", name)
		}
	}
	esm := EmitErrorsESM(nil)
	assertContainsAll(t, esm, []string{
		"class extends Error",
		"Object.setPrototypeOf(this, new.target.prototype)",
		"this.name = tag",
	})
}

func TestEmitErrors_catalogDrivesBothEmits(t *testing.T) {
	esm := EmitErrorsESM(nil)
	dts := EmitErrorsDTS(nil)
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
