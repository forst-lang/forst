package diag_test

import (
	"strings"
	"testing"

	"forst/internal/diag"
)

func TestFormatReport_firstLineStandalone(t *testing.T) {
	t.Parallel()
	out := diag.FormatReport(diag.Report{
		Code:    "result-ok-subject",
		Title:   "Ok() needs a Result",
		Problem: "`is Ok()` only works when the subject is a Result.\nHere the subject has type `String`.",
		Help:    "bind a Result first, then write:\n\n    if r is Ok() { ... }\n    // or\n    ensure r is Ok()",
		Notes:   []string{"`is Ok()` / `is Err()` are Result discriminators, not ordinary type guards."},
	})
	first, _, _ := strings.Cut(out, "\n")
	if first != "error[result-ok-subject]: Ok() needs a Result" {
		t.Fatalf("first line = %q", first)
	}
	if !strings.Contains(out, "help: bind a Result") {
		t.Fatalf("missing help:\n%s", out)
	}
	if !strings.Contains(out, "note: `is Ok()`") {
		t.Fatalf("missing note:\n%s", out)
	}
	if strings.Contains(out, `"fixes"`) || strings.Contains(out, "NewText") {
		t.Fatalf("fixes leaked into text:\n%s", out)
	}
}

func TestFormatReport_shapeUnknownField(t *testing.T) {
	t.Parallel()
	out := diag.FormatReport(diag.Report{
		Code:    "shape-unknown-field",
		Title:   `no field named "nme"`,
		Problem: "Type `User` has no field `nme`.",
		Help:    "did you mean `name`?",
		Notes:   []string{"known fields: name, age, email"},
	})
	if !strings.HasPrefix(out, `error[shape-unknown-field]: no field named "nme"`) {
		t.Fatalf("header:\n%s", out)
	}
	if !strings.Contains(out, "help: did you mean `name`?") {
		t.Fatalf("help:\n%s", out)
	}
}

func TestClosestName(t *testing.T) {
	t.Parallel()
	cands := []string{"Print", "Printf", "Println", "Sprint"}
	if got := diag.ClosestName("Printl", cands); got != "Println" {
		t.Fatalf("Printl → %q", got)
	}
	if got := diag.ClosestName("nme", []string{"name", "age", "email"}); got != "name" {
		t.Fatalf("nme → %q", got)
	}
	if got := diag.ClosestName("x", []string{"ab", "cd"}); got != "" {
		t.Fatalf("ambiguous short → %q", got)
	}
}

func TestFormatKnownList(t *testing.T) {
	t.Parallel()
	got := diag.FormatKnownList("known fields: ", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}, 8)
	if got != "known fields: a, b, c, d, e, f, g, h, …" {
		t.Fatalf("got %q", got)
	}
}
