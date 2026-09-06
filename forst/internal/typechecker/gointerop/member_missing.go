package gointerop

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"

	"forst/internal/ast"
	"forst/internal/diag"
)

// MemberMissingError is returned when an exported Go package member is not found.
// The typechecker converts this into a structured Diagnostic with optional Fixes.
type MemberMissingError struct {
	Span    ast.SourceSpan
	Pkg     string
	Member  string
	Exports []string
}

func (e *MemberMissingError) Error() string {
	if e == nil {
		return ""
	}
	return diag.FormatReport(memberMissingReport(e.Pkg, e.Member, e.Exports))
}

func memberMissingReport(pkg, member string, exports []string) diag.Report {
	problem := fmt.Sprintf("No exported name `%s` in package `%s`.", member, pkg)
	help := "check the spelling or pick an exported name from the package"
	r := diag.Report{
		Code:    "go-member-missing",
		Title:   fmt.Sprintf("%s.%s not found", pkg, member),
		Problem: problem,
		Help:    help,
	}
	if sug := diag.ClosestName(member, exports); sug != "" {
		r.Help = fmt.Sprintf("did you mean `%s`?", sug)
		r.Fixes = []diag.Fix{{Title: "Rename to " + sug, NewText: sug}}
	}
	if note := diag.FormatKnownList("other exports include ", exports, 8); note != "" {
		r.Notes = []string{note}
	}
	return r
}

func exportedNames(scope *types.Scope) []string {
	if scope == nil {
		return nil
	}
	var names []string
	for _, name := range scope.Names() {
		if token.IsExported(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
