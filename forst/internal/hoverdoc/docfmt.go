// Package hoverdoc holds user-facing markdown for LSP hovers (built-in types, guards, keywords).
package hoverdoc

import "strings"

// forstBlock returns a markdown fenced code block labeled for Forst.
func forstBlock(lines ...string) string {
	if len(lines) == 0 {
		return ""
	}
	return "```forst\n" + strings.Join(lines, "\n") + "\n```"
}
