// listen_defaults picks Unix sockets on non-Windows unless the caller opts into TCP.
package invokeserver

import (
	"path/filepath"
	"runtime"
	"strings"

	"forst/internal/unixpath"
)

const (
	envInvokeTransport     = "FORST_INVOKE_TRANSPORT"
	defaultInvokeSocketRel = ".forst/invoke.sock"
	invokeSocketTmpPrefix  = "forst-inv-"
)

// DefaultInvokeSocketPath returns boundaryRoot/.forst/invoke.sock, shortened when needed.
func DefaultInvokeSocketPath(boundaryRoot string) string {
	if boundaryRoot == "" {
		return ""
	}
	return unixpath.EnsureLength(
		filepath.Join(boundaryRoot, defaultInvokeSocketRel),
		invokeSocketTmpPrefix,
	)
}

// preferUnixTransport reports whether invoke should default to a Unix domain socket.
func preferUnixTransport() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	v := strings.ToLower(strings.TrimSpace(lookupEnv(envInvokeTransport)))
	if v == "tcp" || v == "http" {
		return false
	}
	return true
}

// ApplyListenDefaults sets Transport and SocketPath when unset.
// Non-Windows defaults to unix under boundaryRoot/.forst/invoke.sock.
// Set FORST_INVOKE_TRANSPORT=tcp (or Transport: "tcp") to keep loopback TCP.
func ApplyListenDefaults(cfg *Config, boundaryRoot string) {
	if cfg == nil {
		return
	}
	if cfg.BoundaryRoot == "" && boundaryRoot != "" {
		cfg.BoundaryRoot = boundaryRoot
	}
	root := cfg.BoundaryRoot
	if root == "" {
		root = boundaryRoot
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case transportTCP, "http":
		cfg.Transport = transportTCP
		return
	case transportUnix:
		cfg.Transport = transportUnix
		if cfg.SocketPath == "" && root != "" {
			cfg.SocketPath = DefaultInvokeSocketPath(root)
		}
		return
	}

	if preferUnixTransport() && root != "" {
		cfg.Transport = transportUnix
		if cfg.SocketPath == "" {
			cfg.SocketPath = DefaultInvokeSocketPath(root)
		}
		return
	}
	cfg.Transport = transportTCP
}
