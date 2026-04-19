package discovery

import (
	"strings"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

// StreamingSupported reports whether a public function may be invoked with streaming (NDJSON over HTTP).
func StreamingSupported(fn *ast.FunctionNode, tc *typechecker.TypeChecker) bool {
	if fn == nil {
		return false
	}
	if typechecker.IsChannelReturn(fn, tc) {
		return true
	}
	// Check function name for streaming indicators
	name := strings.ToLower(string(fn.Ident.ID))
	streamingKeywords := []string{"stream", "process", "batch", "pipeline"}

	for _, keyword := range streamingKeywords {
		if strings.Contains(name, keyword) {
			return true
		}
	}

	// Check return type for streaming indicators
	var returnTypes []ast.TypeNode
	if tc != nil {
		if sig, exists := tc.Functions[fn.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
			returnTypes = sig.ReturnTypes
		} else {
			returnTypes = fn.ReturnTypes
		}
	} else {
		returnTypes = fn.ReturnTypes
	}

	if len(returnTypes) > 0 {
		// Check the original type name before it gets converted to hash-based names
		originalTypeName := string(returnTypes[0].Ident)
		if strings.Contains(strings.ToLower(originalTypeName), "stream") ||
			strings.Contains(strings.ToLower(originalTypeName), "channel") {
			return true
		}
	}

	return false
}

// analyzeStreamingSupport determines if a function supports streaming
func (d *Discoverer) analyzeStreamingSupport(fn *ast.FunctionNode, tc *typechecker.TypeChecker) bool {
	return StreamingSupported(fn, tc)
}
