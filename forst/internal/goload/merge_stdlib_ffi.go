package goload

import "forst/gateway"

// IsMergeStdlibUserDefinedImport reports whether go/types named types from this Go import path are
// surfaced in Forst as qualified UserDefined types (stable FFI / codegen), rather than implicit
// structural hashes. Covers merge-path stdlib under module "forst" (e.g. forst/gateway).
func IsMergeStdlibUserDefinedImport(importPath string) bool {
	switch importPath {
	case gateway.StdlibImportPath:
		return true
	default:
		return false
	}
}
