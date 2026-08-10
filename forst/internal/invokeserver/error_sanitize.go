package invokeserver

import (
	"path/filepath"
	"strings"
)

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
