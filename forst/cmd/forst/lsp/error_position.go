package lsp

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

var (
	// Legacy string panics from parseErrorMessage() — "Parse error in file:line:col ..."
	reParseErrorFileLineCol = regexp.MustCompile(`Parse error in [^:]+:(\d+):(\d+)`)
	// ParseError.Error() — "Parse error at file:line:col: ..." (Location() is file:line:col)
	reParseErrorAtLineCol = regexp.MustCompile(`Parse error at ([^:]+):(\d+):(\d+):`)
	reFunctionName          = regexp.MustCompile(`(?:function|of function)\s+([A-Za-z_][\w]*)`)
	reTypeFieldCtx          = regexp.MustCompile(`type\s+([A-Za-z_][\w]*)\s+field`)
	reParamCtx              = regexp.MustCompile(`function\s+([A-Za-z_][\w]*)\s+parameter`)
	reReturnCtx             = regexp.MustCompile(`function\s+([A-Za-z_][\w]*)\s+return`)
	reUndefinedSymbol       = regexp.MustCompile(`undefined symbol:\s*(\S+)`)
	reUndefinedFunction     = regexp.MustCompile(`undefined function:\s*(\S+)`)
	reUnknownIdentifier     = regexp.MustCompile(`unknown identifier:\s*(\S+)`)
)

// DiagnosticFromParseError builds an LSP diagnostic at the failing token (lexer uses 1-based line/column).
func DiagnosticFromParseError(fileURI string, pe *parser.ParseError) LSPDiagnostic {
	line1 := pe.Token.Line
	col1 := pe.Token.Column
	if line1 < 1 {
		line1 = 1
	}
	if col1 < 1 {
		col1 = 1
	}
	line0 := lspLineIndex(line1)
	startC := col1 - 1
	endC := startC
	if pe.Token.Value != "" {
		endC = startC + utf8.RuneCountInString(pe.Token.Value)
	} else {
		endC = startC + 1
	}
	msg := pe.Msg
	if msg == "" {
		msg = pe.Error()
	}
	return LSPDiagnostic{
		Range: LSPRange{
			Start: LSPPosition{Line: line0, Character: startC},
			End:   LSPPosition{Line: line0, Character: endC},
		},
		Severity: LSPDiagnosticSeverityError,
		Source:   "forst-parser",
		Code:     ErrorCodeInvalidSyntax,
		Message:  msg,
	}
}

// diagnosticForParseFailure handles *ParseError, legacy string panics, or a generic fallback.
func diagnosticForParseFailure(fileURI, content string, err error) LSPDiagnostic {
	var pe *parser.ParseError
	if errors.As(err, &pe) {
		return DiagnosticFromParseError(fileURI, pe)
	}
	if line1, col1, ok := parseErrorLocationFromMessage(err.Error()); ok {
		return simpleDiagnosticOnLine(fileURI, line1, col1, fmt.Sprintf("Parsing error: %v", err), "forst-parser", ErrorCodeInvalidSyntax)
	}
	line1, col1 := bestEffortLineColumnFromErrorMessage(content, err.Error())
	return simpleDiagnosticOnLine(fileURI, line1, col1, fmt.Sprintf("Parsing error: %v", err), "forst-parser", ErrorCodeInvalidSyntax)
}

func parseErrorLocationFromMessage(msg string) (line1, col1 int, ok bool) {
	if m := reParseErrorFileLineCol.FindStringSubmatch(msg); len(m) >= 3 {
		return atoiLineCol(m[1], m[2])
	}
	// "Parse error at <file>:line:col:" — groups are file, line, column
	if m := reParseErrorAtLineCol.FindStringSubmatch(msg); len(m) >= 4 {
		return atoiLineCol(m[2], m[3])
	}
	return 0, 0, false
}

func atoiLineCol(lineStr, colStr string) (line1, col1 int, ok bool) {
	l, err1 := strconv.Atoi(lineStr)
	c, err2 := strconv.Atoi(colStr)
	if err1 != nil || err2 != nil || l < 1 {
		return 0, 0, false
	}
	if c < 1 {
		c = 1
	}
	return l, c, true
}

// diagnosticForTypecheckOrTransform places diagnostics on the best-effort line derived from error text and source.
func diagnosticForTypecheckOrTransform(fileURI, content string, err error, source, code string) LSPDiagnostic {
	msg := err.Error()
	line1, col1 := bestEffortLineColumnFromErrorMessage(content, msg)
	return simpleDiagnosticOnLine(fileURI, line1, col1, msg, source, code)
}

// diagnosticForTypecheckError prefers a structured typechecker.Diagnostic span; otherwise falls back to heuristics.
func diagnosticForTypecheckError(fileURI, content string, err error, source, defaultCode string) LSPDiagnostic {
	var diag *typechecker.Diagnostic
	if errors.As(err, &diag) && diag != nil {
		msg := diag.Msg
		code := diag.Code
		if code == "" {
			code = defaultCode
		}
		if diag.Span.IsSet() {
			return lspDiagnosticFromASTSpan(fileURI, msg, source, code, diag.Span)
		}
		line1, col1 := bestEffortLineColumnFromErrorMessage(content, msg)
		return simpleDiagnosticOnLine(fileURI, line1, col1, msg, source, code)
	}
	return diagnosticForTypecheckOrTransform(fileURI, content, err, source, defaultCode)
}

// lspDiagnosticFromASTSpan maps a 1-based half-open ast.SourceSpan to an LSP range (0-based; character = UTF-8 byte offset within line for ASCII-aligned lexer columns).
func lspDiagnosticFromASTSpan(_ string, msg, source, code string, span ast.SourceSpan) LSPDiagnostic {
	return LSPDiagnostic{
		Range: LSPRange{
			Start: LSPPosition{Line: lspLineIndex(span.StartLine), Character: span.StartCol - 1},
			End:   LSPPosition{Line: lspLineIndex(span.EndLine), Character: span.EndCol - 1},
		},
		Severity: LSPDiagnosticSeverityError,
		Source:   source,
		Code:     code,
		Message:  msg,
	}
}

func simpleDiagnosticOnLine(fileURI string, line1, col1 int, message, source, code string) LSPDiagnostic {
	if line1 < 1 {
		line1 = 1
	}
	if col1 < 1 {
		col1 = 1
	}
	line0 := lspLineIndex(line1)
	startC := col1 - 1
	// Span to end of line (cap) — LSP clients clip; avoid character 100 hack.
	endC := startC + 120
	return LSPDiagnostic{
		Range: LSPRange{
			Start: LSPPosition{Line: line0, Character: startC},
			End:   LSPPosition{Line: line0, Character: endC},
		},
		Severity: LSPDiagnosticSeverityError,
		Source:   source,
		Code:     code,
		Message:  message,
	}
}

func bestEffortLineColumnFromErrorMessage(content, errMsg string) (line1, col1 int) {
	line1, col1 = 1, 1
	if content == "" {
		return
	}
	// Typechecker: "undefined symbol: name [scope: ...]" — no file:line in the error text
	if m := reUndefinedSymbol.FindStringSubmatch(errMsg); len(m) > 1 {
		return lineAndColumnOfFirstOccurrence(content, m[1])
	}
	if m := reUndefinedFunction.FindStringSubmatch(errMsg); len(m) > 1 {
		return lineAndColumnOfFirstOccurrence(content, m[1])
	}
	if m := reUnknownIdentifier.FindStringSubmatch(errMsg); len(m) > 1 {
		return lineAndColumnOfFirstOccurrence(content, m[1])
	}
	// function Name / of function Name
	if m := reFunctionName.FindStringSubmatch(errMsg); len(m) > 1 {
		if ln := findLineContaining(content, "func "+m[1]); ln > 0 {
			return ln, 1
		}
	}
	if m := reParamCtx.FindStringSubmatch(errMsg); len(m) > 1 {
		if ln := findLineContaining(content, "func "+m[1]); ln > 0 {
			return ln, 1
		}
	}
	if m := reReturnCtx.FindStringSubmatch(errMsg); len(m) > 1 {
		if ln := findLineContaining(content, "func "+m[1]); ln > 0 {
			return ln, 1
		}
	}
	// type SomeType field "x"
	if m := reTypeFieldCtx.FindStringSubmatch(errMsg); len(m) > 1 {
		name := m[1]
		if ln := findLineContaining(content, "type "+name); ln > 0 {
			return ln, 1
		}
		if ln := findLineContaining(content, "type "+name+" ="); ln > 0 {
			return ln, 1
		}
	}
	// undeclared variable 'foo' — first line mentioning the identifier
	if m := regexp.MustCompile(`variable '([^']+)'`).FindStringSubmatch(errMsg); len(m) > 1 {
		return lineAndColumnOfFirstOccurrence(content, m[1])
	}
	return
}

// lineAndColumnOfFirstOccurrence returns 1-based line/column of the first substring match, or (1,1) if absent.
func lineAndColumnOfFirstOccurrence(content, needle string) (line1, col1 int) {
	line1, col1 = 1, 1
	if needle == "" {
		return
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		idx := strings.Index(line, needle)
		if idx >= 0 {
			return i + 1, idx + 1
		}
	}
	return
}

func findLineContaining(content, needle string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, needle) {
			return i + 1
		}
	}
	return 0
}
