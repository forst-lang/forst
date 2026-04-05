package lsp

import (
	"path/filepath"
	"runtime"
	"strings"
)

// filePathFromDocumentURI converts an LSP file:// URI to a host filesystem path.
func filePathFromDocumentURI(uri string) string {
	p := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, "/")
	}
	return p
}

// fileURIForLocalPath builds a file:// URI from an absolute or clean local filesystem path.
func fileURIForLocalPath(path string) string {
	path = filepath.Clean(path)
	s := filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		return "file:///" + s
	}
	return "file://" + s
}

// isForstDocumentURI reports whether uri refers to a Forst source buffer (file:// … .ft).
func isForstDocumentURI(uri string) bool {
	return strings.HasPrefix(uri, "file://") && strings.HasSuffix(uri, ".ft")
}
