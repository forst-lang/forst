package importlocal

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

var moduleExtensions = []string{".tsx", ".ts", ".jsx", ".js", ".mjs", ".cjs"}

// IsValidIdentifierSyntax reports whether name is a valid Forst identifier.
func IsValidIdentifierSyntax(name string) bool {
	if name == "" {
		return false
	}
	r, w := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}
	for _, r := range name[w:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

// DefaultLocalFromModuleID derives the implicit import local from a resolved module id.
func DefaultLocalFromModuleID(moduleID string) string {
	id := strings.ReplaceAll(moduleID, `\`, "/")
	segment := id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		segment = id[i+1:]
	}
	return stripModuleExtension(segment)
}

func stripModuleExtension(name string) string {
	lower := strings.ToLower(name)
	for _, ext := range moduleExtensions {
		if strings.HasSuffix(lower, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

func sanitizeSegment(segment string) string {
	var b strings.Builder
	for _, r := range segment {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			last, _ := utf8.DecodeLastRuneInString(b.String())
			if last != '_' {
				b.WriteRune('_')
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "pkg"
	}
	r, _ := utf8.DecodeRuneInString(out)
	if unicode.IsDigit(r) {
		out = "_" + out
	}
	return out
}
