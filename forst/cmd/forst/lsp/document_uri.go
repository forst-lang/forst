package lsp

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// localPathFromFileURI extracts a host filesystem path from an LSP file:// URI.
// It returns false when the URI is not a usable file URI (wrong scheme, empty path).
func localPathFromFileURI(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" {
		return legacyLocalPathFromFileURI(uri)
	}

	path := u.Path
	if path == "" && u.Opaque != "" {
		path = u.Opaque
	}

	switch runtime.GOOS {
	case "windows":
		host := strings.ToLower(u.Host)
		if host != "" && host != "localhost" {
			// UNC: file://server/share/path
			share := strings.TrimPrefix(path, "/")
			share = strings.ReplaceAll(share, "/", `\`)
			return filepath.Clean(`\\` + u.Host + `\` + share), true
		}
		if len(path) >= 2 && path[0] == '/' && path[2] == ':' {
			path = path[1:] // /C:/... -> C:/...
		}
	default:
	}

	path = filepath.FromSlash(path)
	if path == "" {
		return legacyLocalPathFromFileURI(uri)
	}
	return path, true
}

func legacyLocalPathFromFileURI(uri string) (string, bool) {
	if !strings.HasPrefix(uri, "file://") {
		return "", false
	}
	p := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		p = strings.TrimPrefix(p, "/")
	}
	p = filepath.FromSlash(p)
	if p == "" {
		return "", false
	}
	return p, true
}

// canonicalDirForPath returns the canonical directory containing path (absolute, symlink-resolved when possible).
func canonicalDirForPath(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Dir(canonicalLocalPath(path))
}

// canonicalLocalPath returns an absolute, cleaned filesystem path.
// We intentionally do not EvalSymlinks here: LSP clients and tests expect stable round-trips with
// filepath.Abs, and symlink targets like /private/var on macOS would otherwise diverge from editor paths.
func canonicalLocalPath(path string) string {
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" && (strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, `//`)) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(abs)
}

// filePathFromDocumentURI converts an LSP file:// URI to a host filesystem path suitable for I/O.
func filePathFromDocumentURI(uri string) string {
	if p, ok := localPathFromFileURI(uri); ok && p != "" {
		return canonicalLocalPath(p)
	}
	if strings.HasPrefix(uri, "file://") {
		p := strings.TrimPrefix(uri, "file://")
		if runtime.GOOS == "windows" {
			p = strings.TrimPrefix(p, "/")
		}
		return canonicalLocalPath(filepath.FromSlash(p))
	}
	return ""
}

// canonicalFileURI returns a stable file:// URI for the same file as uri (Abs, Clean, symlink resolve).
// Non-file URIs are returned unchanged.
func canonicalFileURI(uri string) string {
	p, ok := localPathFromFileURI(uri)
	if !ok || p == "" {
		return uri
	}
	return fileURIForLocalPath(p)
}

// fileURIForLocalPath builds a file:// URI from a local filesystem path.
func fileURIForLocalPath(path string) string {
	path = canonicalLocalPath(path)
	s := filepath.ToSlash(path)
	if runtime.GOOS == "windows" {
		return "file:///" + s
	}
	return "file://" + s
}

// isForstDocumentURI reports whether uri refers to a Forst source buffer (file:// … .ft).
func isForstDocumentURI(uri string) bool {
	if p, ok := localPathFromFileURI(uri); ok {
		return strings.HasSuffix(strings.ToLower(p), ".ft")
	}
	return strings.HasPrefix(uri, "file://") && strings.HasSuffix(uri, ".ft")
}
