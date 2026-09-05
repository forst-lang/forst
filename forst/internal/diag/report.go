// Package diag formats structured compiler diagnostics for CLI, LSP, and agents.
package diag

import (
	"fmt"
	"strings"

	"forst/internal/ast"
)

// Report is a structured diagnostic (Elm/Rust/Zig-style) with stable semantic codes.
type Report struct {
	Code    string
	Title   string
	Problem string
	Help    string
	Notes   []string
	Fixes   []Fix // machine-only; not embedded in FormatReport text
}

// Fix is a safe, applyable text edit for IDE quickfixes and coding agents.
type Fix struct {
	Title   string
	NewText string
	// Span overrides the diagnostic primary span when the edit target differs
	// (e.g. rename only the member ident inside pkg.Member).
	Span ast.SourceSpan
}

// FormatReport renders a human- and LLM-friendly multiline diagnostic.
// Fixes are never serialized into the text.
func FormatReport(r Report) string {
	code := strings.TrimSpace(r.Code)
	title := strings.TrimSpace(r.Title)

	if code == "" && title == "" && strings.TrimSpace(r.Problem) == "" && strings.TrimSpace(r.Help) == "" {
		return ""
	}

	var b strings.Builder
	if code != "" {
		if title != "" {
			fmt.Fprintf(&b, "error[%s]: %s", code, title)
		} else {
			fmt.Fprintf(&b, "error[%s]:", code)
		}
	} else if title != "" {
		fmt.Fprintf(&b, "error: %s", title)
	}

	problem := strings.TrimSpace(r.Problem)
	help := strings.TrimSpace(r.Help)
	if problem != "" || help != "" || len(r.Notes) > 0 {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		if problem != "" {
			b.WriteByte('\n')
			writeIndentedBlock(&b, problem)
		}
		if help != "" {
			b.WriteString("\n\n  help: ")
			writeHelpBody(&b, help)
		}
		for _, note := range r.Notes {
			note = strings.TrimSpace(note)
			if note == "" {
				continue
			}
			b.WriteString("\n  note: ")
			writeIndentedContinuation(&b, note, "  note: ")
		}
	}
	return b.String()
}

func writeIndentedBlock(b *strings.Builder, text string) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("  ")
		b.WriteString(strings.TrimRight(line, " \t"))
	}
}

func writeHelpBody(b *strings.Builder, help string) {
	lines := strings.Split(help, "\n")
	if len(lines) == 0 {
		return
	}
	b.WriteString(strings.TrimRight(lines[0], " \t"))
	for _, line := range lines[1:] {
		b.WriteByte('\n')
		trimmed := strings.TrimRight(line, " \t")
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "    ") || strings.HasPrefix(trimmed, "\t") {
			b.WriteString(trimmed)
			continue
		}
		b.WriteString("  ")
		b.WriteString(trimmed)
	}
}

func writeIndentedContinuation(b *strings.Builder, text, label string) {
	lines := strings.Split(text, "\n")
	b.WriteString(strings.TrimRight(lines[0], " \t"))
	pad := strings.Repeat(" ", len(label))
	for _, line := range lines[1:] {
		b.WriteByte('\n')
		b.WriteString(pad)
		b.WriteString(strings.TrimRight(line, " \t"))
	}
}
