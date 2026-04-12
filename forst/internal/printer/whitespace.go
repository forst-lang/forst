package printer //nolint:revive // package name matches internal/printer; overlaps with go/printer

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// UTF16Len returns the number of UTF-16 code units in s (LSP character offsets).
func UTF16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// LSPCharPos is a 0-based line and UTF-16 character offset (LSP text positions).
type LSPCharPos struct {
	Line      int
	Character int
}

// EndPositionExclusive returns the LSP position immediately after the last code unit in src
// (newlines normalized to \n). Used as the exclusive end of a full-document range.
func EndPositionExclusive(src string) LSPCharPos {
	if src == "" {
		return LSPCharPos{Line: 0, Character: 0}
	}
	norm := strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n")
	line := strings.Count(norm, "\n")
	idx := strings.LastIndex(norm, "\n")
	var lastLine string
	if idx == -1 {
		lastLine = norm
	} else {
		lastLine = norm[idx+1:]
	}
	return LSPCharPos{Line: line, Character: UTF16Len(lastLine)}
}

// expandLeadingTabs replaces each leading U+0009 tab with tabSize spaces.
func expandLeadingTabs(line string, tabSize int) string {
	if tabSize <= 0 {
		tabSize = 4
	}
	var b strings.Builder
	i := 0
	for i < len(line) && line[i] == '\t' {
		b.WriteString(strings.Repeat(" ", tabSize))
		i++
	}
	if i == 0 {
		return line
	}
	b.WriteString(line[i:])
	return b.String()
}

// FormatForstWhitespace normalizes CRLF/CR to LF, trims trailing spaces and tabs on each line,
// optionally expands leading tabs to spaces, and appends a final newline when the result is non-empty.
func FormatForstWhitespace(src string, tabSize int, insertSpaces bool) string {
	if tabSize <= 0 {
		tabSize = 4
	}
	norm := strings.ReplaceAll(strings.ReplaceAll(src, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(norm, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if insertSpaces {
			line = expandLeadingTabs(line, tabSize)
		}
		lines[i] = line
	}
	out := strings.Join(lines, "\n")
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out
}

// FormatDocument applies the same pipeline as LSP textDocument/formatting and `forst fmt`:
// pretty-print when the file parses; otherwise whitespace-only normalization. Then applies
// tab/spaces policy. On parse failure, logs at trace level if log is non-nil.
func FormatDocument(src, fileID string, tabSize int, insertSpaces bool, log *logrus.Logger) string {
	pretty, err := FormatSource(src, fileID, log)
	if err != nil {
		if log != nil {
			log.WithError(err).Trace("format: pretty-print skipped, using whitespace-only normalization")
		}
		return FormatForstWhitespace(src, tabSize, insertSpaces)
	}
	return FormatForstWhitespace(pretty, tabSize, insertSpaces)
}
