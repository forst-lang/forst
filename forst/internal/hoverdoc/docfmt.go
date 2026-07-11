// Package hoverdoc holds user-facing markdown for LSP hovers (built-in types, guards, keywords).
package hoverdoc

import "strings"

// Fence language ids used for syntax highlighting in LSP markdown hovers.
const (
	LangForst      = "ft"
	LangGo         = "go"
	LangTypeScript = "typescript"
)

// CodeBlock returns a fenced markdown code block for the given language.
func CodeBlock(lang string, lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	return "```" + lang + "\n" + strings.Join(lines, "\n") + "\n```"
}

// ForstBlock returns a ```ft fenced block.
func ForstBlock(lines ...string) string {
	return CodeBlock(LangForst, lines...)
}

// GoBlock returns a ```go fenced block.
func GoBlock(lines ...string) string {
	return CodeBlock(LangGo, lines...)
}

// TypeScriptBlock returns a ```typescript fenced block.
func TypeScriptBlock(lines ...string) string {
	return CodeBlock(LangTypeScript, lines...)
}

// TypeScriptModuleLine returns a TypeScript-style module path line (no file extension).
func TypeScriptModuleLine(modulePath string) string {
	return `module "` + modulePath + `"`
}

// TypeScriptModuleBlock returns a fenced TypeScript module path hover line.
func TypeScriptModuleBlock(modulePath string) string {
	return TypeScriptBlock(TypeScriptModuleLine(modulePath))
}

// Section returns a bold markdown section title.
func Section(title string) string {
	return "**" + title + "**"
}

// forstBlock is an alias for ForstBlock (package-internal call sites).
func forstBlock(lines ...string) string {
	return ForstBlock(lines...)
}
