package gointerop

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/goload"
)

// FallbackImportLocal derives Go import path and local name from a Forst import line.
func FallbackImportLocal(imp ast.ImportNode) (path, local string) {
	ip := goload.ImportPathFromForst(imp.Path)
	if ip == "" {
		return "", ""
	}
	if imp.Alias != nil {
		return ip, string(imp.Alias.ID)
	}
	if i := strings.LastIndex(ip, "/"); i >= 0 {
		return ip, ip[i+1:]
	}
	return ip, ip
}

// IdentifierExported reports whether name is an exported Go identifier.
func IdentifierExported(name string) bool {
	if name == "" || name[0] == '_' {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return r != utf8.RuneError && unicode.IsUpper(r)
}
