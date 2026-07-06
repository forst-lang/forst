// Compile harness helpers for parse → typecheck → transform → Go emit tests.
package transformergo

import (
	"go/format"
	"testing"

	"forst/internal/generators"
	"forst/internal/modulecheck"
	"forst/internal/testutil"
	"forst/internal/typechecker"
)

// MustCompileGo parses, typechecks, transforms, and generates Go source from src.
func MustCompileGo(tb testing.TB, src string, opts testutil.CompileOpts) string {
	tb.Helper()
	if opts.ModuleProviders {
		moduleRoot := opts.GoWorkspaceDir
		if moduleRoot == "" && opts.UseModuleRoot {
			moduleRoot = testutil.ModuleRoot(tb)
		}
		if moduleRoot == "" {
			tb.Fatal("ModuleProviders requires GoWorkspaceDir or UseModuleRoot")
		}
		log := opts.Logger
		if log == nil {
			log = testutil.TestLogger(tb, nil)
		}
		if _, err := modulecheck.CheckModuleProviders(log, modulecheck.Options{ModuleRoot: moduleRoot}); err != nil {
			tb.Fatalf("module providers: %v", err)
		}
	}
	tc, nodes := typechecker.MustTypecheck(tb, src, opts.TypecheckOpts)
	log := opts.Logger
	if log == nil {
		log = testutil.TestLogger(tb, nil)
	}
	tr := New(tc, log, opts.ExportStructFields)
	goFile, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		tb.Fatalf("transform: %v", err)
	}
	code, err := generators.GenerateGoCode(goFile)
	if err != nil {
		tb.Fatalf("generate Go: %v", err)
	}
	if opts.FormatGo {
		formatted, err := format.Source([]byte(code))
		if err != nil {
			tb.Fatalf("format Go: %v", err)
		}
		code = string(formatted)
	}
	return code
}

// MustCompileMergedGo merges paths, typechecks, transforms, and generates Go source.
func MustCompileMergedGo(tb testing.TB, paths []string, opts testutil.CompileOpts) string {
	tb.Helper()
	tc, nodes := typechecker.MustTypecheckMerged(tb, paths, opts.TypecheckOpts)
	log := opts.Logger
	if log == nil {
		log = testutil.TestLogger(tb, nil)
	}
	tr := New(tc, log, opts.ExportStructFields)
	goFile, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		tb.Fatalf("transform: %v", err)
	}
	code, err := generators.GenerateGoCode(goFile)
	if err != nil {
		tb.Fatalf("generate Go: %v", err)
	}
	if opts.FormatGo {
		formatted, err := format.Source([]byte(code))
		if err != nil {
			tb.Fatalf("format Go: %v", err)
		}
		code = string(formatted)
	}
	return code
}
