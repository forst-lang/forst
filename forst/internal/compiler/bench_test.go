package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func examplePath(b *testing.B, rel string) string {
	b.Helper()
	path := filepath.Join("..", "..", "..", "examples", "in", rel)
	if _, err := os.Stat(path); err != nil {
		b.Fatalf("example file %s: %v", rel, err)
	}
	return path
}

func benchmarkCompileFile(b *testing.B, exampleRel string) {
	b.Helper()
	path := examplePath(b, exampleRel)
	c := New(Args{
		Command:  "run",
		FilePath: path,
		LogLevel: "error",
	}, silentCompilerTestLogger())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.CompileFile(); err != nil {
			b.Fatalf("CompileFile: %v", err)
		}
	}
}

func benchmarkCompileMerged(b *testing.B, rootRel, entryRel string) {
	b.Helper()
	root := filepath.Join("..", "..", "..", "examples", "in", rootRel)
	entry := filepath.Join(root, entryRel)
	if _, err := os.Stat(entry); err != nil {
		b.Fatalf("example entry %s: %v", entry, err)
	}
	c := New(Args{
		Command:     "run",
		FilePath:    entry,
		PackageRoot: root,
		LogLevel:    "error",
	}, silentCompilerTestLogger())
	if _, err := c.CompileFile(); err != nil {
		b.Fatalf("warm CompileFile: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := c.CompileFile(); err != nil {
			b.Fatalf("CompileFile: %v", err)
		}
	}
}

func BenchmarkCompile_basic(b *testing.B) {
	benchmarkCompileFile(b, "basic.ft")
}

func BenchmarkCompile_shapeGuard(b *testing.B) {
	benchmarkCompileFile(b, filepath.Join("rfc", "guard", "shape_guard.ft"))
}

func BenchmarkCompile_generics(b *testing.B) {
	benchmarkCompileFile(b, "generics.ft")
}

func BenchmarkCompile_tictactoe(b *testing.B) {
	benchmarkCompileMerged(b, "tictactoe", "server.ft")
}

func BenchmarkCompile_providers(b *testing.B) {
	benchmarkCompileMerged(b, filepath.Join("rfc", "providers"), "main_wiring.ft")
}
