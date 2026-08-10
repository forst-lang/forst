// error_sanitize strips filesystem paths from user-visible invoke error strings.
package invokeserver

import (
	"path/filepath"
	"strings"
)

func looksLikePathToken(part string) bool {
	return filepath.IsAbs(part) || strings.ContainsAny(part, `/\`)
}

// safeErrorMessage replaces absolute or slash-containing path fragments with [path].
func safeErrorMessage(msg string) string {
	if msg == "" {
		return msg
	}
	if !strings.ContainsAny(msg, `/\`) && !strings.Contains(msg, string(filepath.Separator)) {
		return msg
	}
	var b strings.Builder
	word := strings.Builder{}
	flushWord := func() {
		part := word.String()
		word.Reset()
		if part == "" {
			return
		}
		if looksLikePathToken(part) {
			b.WriteString("[path]")
		} else {
			b.WriteString(part)
		}
	}
	for _, r := range msg {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			flushWord()
			b.WriteRune(r)
		} else {
			word.WriteRune(r)
		}
	}
	flushWord()
	return b.String()
}
