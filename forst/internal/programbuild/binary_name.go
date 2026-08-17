package programbuild

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BinaryFileName derives the linked executable name from the entry .ft stem (Go-like).
func BinaryFileName(entryPath, goos string) (string, error) {
	if strings.TrimSpace(entryPath) == "" {
		return "", fmt.Errorf("forst build: missing entry path for binary name")
	}
	stem := strings.TrimSuffix(filepath.Base(entryPath), filepath.Ext(entryPath))
	stem = sanitizeBinaryStem(stem)
	if stem == "" {
		stem = "main"
	}
	if goos == "windows" {
		return stem + ".exe", nil
	}
	return stem, nil
}

func sanitizeBinaryStem(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == '.' || r == ' ' {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
