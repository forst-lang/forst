package hoverdoc

import (
	"strings"

	"forst/internal/ast"
)

// Built-in assertion / guard names (ensure and if … is …) plus Result discriminators.
const (
	GuardMin          = "Min"
	GuardMax          = "Max"
	GuardLessThan     = "LessThan"
	GuardGreaterThan  = "GreaterThan"
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

func guardDoc(title, body string) string {
	var b strings.Builder
	b.WriteString("**`")
	b.WriteString(title)
	b.WriteString("`** (guard)\n\n")
	b.WriteString(body)
	return b.String()
}

var guardDocs = map[string]string{
	GuardMin: guardDoc(GuardMin,
		"Lower bound. For strings and arrays, compares `len(value)` to the integer argument; for `Int` and `Float`, compares the numeric value. Used in `ensure x is Min(n)` or `if x is Min(n)`."),
	GuardMax: guardDoc(GuardMax,
		"Upper bound. For strings and arrays, compares `len(value)` to the integer argument; for `Int` and `Float`, compares the numeric value."),
	GuardLessThan: guardDoc(GuardLessThan,
		"For integers and floats, requires `value < n` (implemented with `>=` on the right-hand literal in generated checks)."),
	GuardGreaterThan: guardDoc(GuardGreaterThan,
		"For integers and floats, requires `value > n` (implemented with `<=` on the right-hand literal in generated checks)."),
	GuardHasPrefix: guardDoc(GuardHasPrefix,
		"For strings, requires `strings.HasPrefix(value, prefix)` to hold for the given string literal argument."),
	GuardContains: guardDoc(GuardContains,
		"For strings, requires `strings.Contains(value, substr)` to hold for the given string literal argument."),
	GuardTrue: guardDoc(GuardTrue,
		"For booleans, requires the value to be `true`."),
	GuardFalse: guardDoc(GuardFalse,
		"For booleans, requires the value to be `false`."),
	GuardNil: guardDoc(GuardNil,
		"For pointers and `Error`, requires the value to be nil (or an untyped nil in Go terms)."),
	GuardPresent: guardDoc(GuardPresent,
		"For pointers and `Error`, requires the value to be non-nil (`!= nil`)."),
	GuardNotEmpty: guardDoc(GuardNotEmpty,
		"For strings and arrays, requires non-zero length (`len != 0`)."),
	GuardValid: guardDoc(GuardValid,
		"Placeholder hook for validation. The compiler may emit a conservative check; prefer explicit predicates or user-defined type guards for real validation."),
	GuardValue: guardDoc(GuardValue,
		"Refines a type to a single compile-time value (dependent-style literal typing)."),
	GuardMatch: guardDoc(GuardMatch,
		"Structural matching against a shape or pattern in assertions (see type checker rules for the surrounding type)."),
	GuardOk: guardDoc(GuardOk,
		"For `Result(S, F)`, narrows to the success payload. In `if x is Ok()` the then-branch sees `S`; `ensure x is Ok()` narrows following statements."),
	GuardErr: guardDoc(GuardErr,
		"For `Result(S, F)`, narrows to the failure side. In `if x is Err()` the then-branch sees the failure type (typically error-kinded)."),
}
