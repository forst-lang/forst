package goload

import (
	"testing"

	"forst/gateway"
)

func TestIsMergeStdlibUserDefinedImport(t *testing.T) {
	if !IsMergeStdlibUserDefinedImport(gateway.StdlibImportPath) {
		t.Fatal("expected gateway stdlib path")
	}
	if IsMergeStdlibUserDefinedImport("strings") {
		t.Fatal("expected false for std library")
	}
}
