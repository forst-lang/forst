// Package invokeembed exposes embedded invoke server hooks for generated app sandboxes.
package invokeembed

import (
	"forst/internal/invokedispatch"
	"forst/internal/invokeserver"
)

// FunctionMeta is invoke handler metadata for generated companions.
type FunctionMeta = invokedispatch.FunctionMeta

// MustStartEmbedded starts the embedded invoke HTTP server.
func MustStartEmbedded() {
	invokeserver.MustStartEmbedded()
}

// WaitForShutdown blocks until SIGINT/SIGTERM for embedded invoke binaries.
func WaitForShutdown() {
	invokeserver.WaitForShutdown()
}

// GlobalRegistry returns the invoke handler registry populated by generated init code.
func GlobalRegistry() *invokedispatch.Registry {
	return invokeserver.GlobalRegistry()
}
