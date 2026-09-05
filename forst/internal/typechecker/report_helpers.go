package typechecker

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"

	"forst/internal/ast"
	"forst/internal/diag"
)

func resultOkSubjectError(disc, gotType string, span ast.SourceSpan) error {
	code := "result-ok-subject"
	title := disc + "() needs a Result"
	if disc == "Err" {
		code = "result-err-subject"
		title = "Err() needs a Result"
	}
	return reportf(span, code, title,
		fmt.Sprintf("`is %s()` only works when the subject is a Result.\nHere the subject has type `%s`.", disc, gotType),
		"bind a Result first, then write:\n\n    if r is "+disc+"() { ... }\n    // or\n    ensure r is "+disc+"()",
		"`is Ok()` / `is Err()` are Result discriminators, not ordinary type guards.")
}

func guardUndefinedError(name string, span ast.SourceSpan) error {
	return reportf(span, "guard-undefined", fmt.Sprintf("type guard `%s` not found", name),
		fmt.Sprintf("No `is (…) %s()` (and no matching builtin) is in scope.", name),
		fmt.Sprintf("declare the guard, or use a builtin like `Present()` / `Ok()`:\n\n    is (user User) %s() {\n        ensure user.age is Min(18)\n    }", name))
}

func resultOkSubjectPlaceError(span ast.SourceSpan) error {
	return reportf(span, "result-ok-subject-place", "Ok/Err needs a simple name",
		"The subject of `is Ok()` / `is Err()` must be a variable (or field path), not a bare call or literal.",
		"bind it first:\n\n    r := fetch()\n    if r is Ok() { ... }")
}

func shapeUnknownFieldError(typ ast.TypeIdent, field string, known []string, span ast.SourceSpan) error {
	typeName := ""
	if typ != "" {
		typeName = formatTypeIdentForDiag(typ)
	}
	problem := fmt.Sprintf("Type `%s` has no field `%s`.", typeName, field)
	if typeName == "" {
		problem = fmt.Sprintf("This shape has no field `%s`.", field)
	}
	help := "check the spelling or add the field to the shape"
	var fixes []diag.Fix
	if sug := diag.ClosestName(field, known); sug != "" {
		help = fmt.Sprintf("did you mean `%s`?", sug)
		fixes = []diag.Fix{{Title: "Rename to " + sug, NewText: sug, Span: span}}
	}
	var notes []string
	if note := diag.FormatKnownList("known fields: ", known, 8); note != "" {
		notes = append(notes, note)
	}
	title := fmt.Sprintf("no field named %q", field)
	if len(fixes) > 0 {
		return reportWithFixes(span, "shape-unknown-field", title, problem, help, fixes, notes...)
	}
	return reportf(span, "shape-unknown-field", title, problem, help, notes...)
}

func goMemberMissingError(pkg, member string, exports []string, span ast.SourceSpan) error {
	problem := fmt.Sprintf("No exported name `%s` in package `%s`.", member, pkg)
	help := "check the spelling or pick an exported name from the package"
	var fixes []diag.Fix
	if sug := diag.ClosestName(member, exports); sug != "" {
		help = fmt.Sprintf("did you mean `%s`?", sug)
		fixes = []diag.Fix{{Title: "Rename to " + sug, NewText: sug, Span: span}}
	}
	var notes []string
	if note := diag.FormatKnownList("other exports include ", exports, 8); note != "" {
		notes = append(notes, note)
	}
	title := fmt.Sprintf("%s.%s not found", pkg, member)
	if len(fixes) > 0 {
		return reportWithFixes(span, "go-member-missing", title, problem, help, fixes, notes...)
	}
	return reportf(span, "go-member-missing", title, problem, help, notes...)
}

func shapeFieldNames(fields map[string]ast.ShapeFieldNode) []string {
	if len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func goExportedNames(scope *types.Scope) []string {
	if scope == nil {
		return nil
	}
	var names []string
	for _, name := range scope.Names() {
		if !token.IsExported(name) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
