package hoverdoc

import (
	"strings"

	"forst/internal/ast"
)

// Built-in assertion / guard names (ensure and if … is …) plus Result discriminators.
const (
	GuardMin           = "Min"
	GuardMax           = "Max"
	GuardLessThan      = "LessThan"
	GuardGreaterThan   = "GreaterThan"
	GuardHasPrefix     = "HasPrefix"
	GuardContains      = "Contains"
	GuardTrue          = "True"
	GuardFalse         = "False"
	GuardNil           = "Nil"
	GuardPresent       = "Present"
	GuardNotEmpty      = "NotEmpty"
	GuardValid         = "Valid"
	GuardValue         = ast.ValueConstraint // "Value"
	GuardMatch         = "Match"
	GuardOk            = "Ok"
	GuardErr           = "Err"
)

// BuiltinGuardNames lists every name with an entry in guardDocs (for tests and tooling).
var BuiltinGuardNames = []string{
	GuardMin, GuardMax, GuardLessThan, GuardGreaterThan,
	GuardHasPrefix, GuardContains,
	GuardTrue, GuardFalse,
	GuardNil, GuardPresent,
	GuardNotEmpty, GuardValid,
	GuardValue, GuardMatch,
	GuardOk, GuardErr,
}

// GuardMarkdown returns markdown describing a built-in guard or Result discriminator, or ""
// if the name is not a documented built-in (user-defined type guards are intentionally omitted).
func GuardMarkdown(name string) string {
	return guardDocs[name]
}

// GuardMarkdownQualified is like GuardMarkdown, but when baseTypeDisplay is non-empty the title line
// uses **`<Base>.<name>`** (guard) (e.g. Int.GreaterThan) instead of **`<name>`** (guard).
func GuardMarkdownQualified(baseTypeDisplay, name string) string {
	doc := GuardMarkdown(name)
	if doc == "" {
		return ""
	}
	base := strings.TrimSpace(baseTypeDisplay)
	if base == "" {
		return doc
	}
	oldPrefix := "**`" + name + "`** (guard)\n\n"
	newPrefix := "**`" + base + "." + name + "`** (guard)\n\n"
	if strings.HasPrefix(doc, oldPrefix) {
		return newPrefix + doc[len(oldPrefix):]
	}
	return doc
}

func guardDoc(title, body string) string {
	var b strings.Builder
	b.WriteString("**`")
	b.WriteString(title)
	b.WriteString("`** (guard)\n\n")
	b.WriteString(body)
	return b.String()
}

var guardDocs = map[string]string{
	GuardMin: guardDoc(GuardMin, strings.Join([]string{
		"Requires a **minimum**: for `String` and `Array`, compares `len` to the integer argument; for `Int` / `Float`, compares the numeric value.",
		"",
		"**Example**",
		"",
		forstBlock(
			"ensure name is Min(1)",
			"ensure scores is Min(0)",
		),
	}, "\n")),
	GuardMax: guardDoc(GuardMax, strings.Join([]string{
		"Requires a **maximum** (same length vs value rules as `Min`).",
		"",
		"**Example**",
		"",
		forstBlock("ensure label is Max(80)"),
	}, "\n")),
	GuardLessThan: guardDoc(GuardLessThan, strings.Join([]string{
		"For `Int` / `Float`, requires the value to stay **strictly below** the given number (the compiler emits the matching comparison).",
		"",
		"**Example**",
		"",
		forstBlock("ensure n is LessThan(100)"),
	}, "\n")),
	GuardGreaterThan: guardDoc(GuardGreaterThan, strings.Join([]string{
		"For `Int` / `Float`, requires the value to stay **strictly above** the given number.",
		"",
		"**Example**",
		"",
		forstBlock("ensure n is GreaterThan(0)"),
	}, "\n")),
	GuardHasPrefix: guardDoc(GuardHasPrefix, strings.Join([]string{
		"For `String`, requires the value to start with the given **string literal** (lowers to `strings.HasPrefix`).",
		"",
		"**Example**",
		"",
		forstBlock(`ensure url is HasPrefix("https://")`),
	}, "\n")),
	GuardContains: guardDoc(GuardContains, strings.Join([]string{
		"For `String`, requires the substring to appear (`strings.Contains`).",
		"",
		"**Example**",
		"",
		forstBlock(`ensure msg is Contains("error")`),
	}, "\n")),
	GuardTrue: guardDoc(GuardTrue, strings.Join([]string{
		"For `Bool`, requires the value to be **true**.",
		"",
		"**Example**",
		"",
		forstBlock("ensure ok is True()"),
	}, "\n")),
	GuardFalse: guardDoc(GuardFalse, strings.Join([]string{
		"For `Bool`, requires the value to be **false**.",
		"",
		"**Example**",
		"",
		forstBlock("ensure done is False()"),
	}, "\n")),
	GuardNil: guardDoc(GuardNil, strings.Join([]string{
		"For pointers and `Error`, requires **nil**.",
		"",
		"**Example**",
		"",
		forstBlock("ensure err is Nil()"),
	}, "\n")),
	GuardPresent: guardDoc(GuardPresent, strings.Join([]string{
		"For pointers and `Error`, requires **non-nil** (`!= nil`).",
		"",
		"**Example**",
		"",
		forstBlock("ensure err is Present()"),
	}, "\n")),
	GuardNotEmpty: guardDoc(GuardNotEmpty, strings.Join([]string{
		"For `String` and `Array`, requires `len != 0`.",
		"",
		"**Example**",
		"",
		forstBlock("ensure line is NotEmpty()"),
	}, "\n")),
	GuardValid: guardDoc(GuardValid, strings.Join([]string{
		"Reserved hook for custom validation. The compiler may emit a placeholder check—use explicit guards or a user-defined type guard for real rules.",
	}, "\n")),
	GuardValue: guardDoc(GuardValue, strings.Join([]string{
		"Refines a type to a **single compile-time value** (literal typing).",
		"",
		"**Example**",
		"",
		forstBlock("// appears in assertion-style types as Value(...)"),
	}, "\n")),
	GuardMatch: guardDoc(GuardMatch, strings.Join([]string{
		"Structural match against a shape or pattern inside an assertion—see the surrounding type for what is compared.",
	}, "\n")),
	GuardOk: guardDoc(GuardOk, strings.Join([]string{
		"For `Result(S, F)`, picks the **success** side. In `if x is Ok()` the then-branch sees `S`; after `ensure x is Ok()`, later code is narrowed.",
		"",
		"**Example**",
		"",
		forstBlock(
			"if r is Ok() {",
			"  // r is the success payload here",
			"}",
		),
	}, "\n")),
	GuardErr: guardDoc(GuardErr, strings.Join([]string{
		"For `Result(S, F)`, picks the **failure** side (often error-like).",
		"",
		"**Example**",
		"",
		forstBlock(
			"if r is Err() {",
			"  // handle failure",
			"}",
		),
	}, "\n")),
}
