// error_sanitize strips filesystem paths from user-visible invoke error strings.
package invokeserver

import (
	"path/filepath"
	"strings"
)

// safeErrorMessage replaces absolute or slash-containing path fragments with [path].
func safeErrorMessage(msg string) string {
	if msg == "" {
		return msg
	}
	if strings.Contains(msg, string(filepath.Separator)) {
		parts := strings.Fields(msg)
		for i, part := range parts {
			if filepath.IsAbs(part) || strings.Contains(part, "/") || strings.Contains(part, `\`) {
				parts[i] = "[path]"
			}
		}
		return strings.Join(parts, " ")
	}
	return msg
}
