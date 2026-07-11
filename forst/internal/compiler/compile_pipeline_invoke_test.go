package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompile_embeddedInvoke_emitsCompanion(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "in", "rfc", "embedded-invoke")
	mainPath := filepath.Join(root, "main.ft")
	if _, err := os.Stat(mainPath); err != nil {
		t.Skip("embedded-invoke example not present:", err)
	}
	c := New(Args{
		Command:     "build",
		FilePath:    mainPath,
		PackageRoot: root,
		LogLevel:    "error",
	}, nil)
	_, _, invokeCode, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	for _, want := range []string{
		"invokeserver.MustStartEmbedded",
		"forst_invoke_main_Echo",
		"ForstInvokeWaitForShutdown",
	} {
		if !strings.Contains(invokeCode, want) {
			t.Fatalf("missing %q in invoke companion:\n%s", want, invokeCode)
		}
	}
}

func TestCompile_remixServe_embeddedAndHostMode(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "in", "rfc", "node-interop", "remix-serve")
	mainPath := filepath.Join(root, "main.ft")
	if _, err := os.Stat(mainPath); err != nil {
		t.Skip("remix-serve example not present:", err)
	}
	c := New(Args{
		Command:            "build",
		FilePath:           mainPath,
		PackageRoot:        root,
		ExportStructFields: true,
		LogLevel:           "error",
	}, nil)
	_, nodeRuntime, invokeCode, err := c.CompileWithNodeRuntime()
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if nodeRuntime == "" {
		t.Fatal("expected node runtime companion")
	}
	if invokeCode == "" {
		t.Fatal("expected invoke server companion")
	}
	for _, want := range []string{
		"forstNodeManifestJSON",
		"nodert.CallSync",
		"invokeserver.MustStartEmbedded",
		"forst_invoke_main_ListTodos",
	} {
		body := nodeRuntime + invokeCode
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in companions", want)
		}
	}
}
