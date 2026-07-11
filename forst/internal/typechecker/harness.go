// Test harness helpers for parse → typecheck. Used from tests across the module.
package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/ftconfig"
	"forst/internal/goload"
	"forst/internal/parser"
	"forst/internal/testutil"

	"github.com/sirupsen/logrus"
)

func applyTypecheckOpts(tb testing.TB, tc *TypeChecker, opts testutil.TypecheckOpts) {
	tb.Helper()
	switch {
	case opts.GoWorkspaceDir != "":
		tc.GoWorkspaceDir = opts.GoWorkspaceDir
	case opts.UseModuleRoot:
		tc.GoWorkspaceDir = testutil.ModuleRoot(tb)
	}
	if opts.NodeBoundaryRoot != "" {
		tc.NodeBoundaryRoot = opts.NodeBoundaryRoot
	}
	if opts.ForstFileDir != "" {
		tc.ForstFileDir = opts.ForstFileDir
	}
	if opts.NodeImportPolicy != "" {
		tc.NodeImportPolicy = opts.NodeImportPolicy
	} else if opts.ForstFileDir != "" {
		tc.NodeImportPolicy = ftconfig.ImportPolicyFromDir(opts.ForstFileDir)
	}
	if opts.SamePackageGoImport != "" {
		tc.SetSamePackageGoImportPath(opts.SamePackageGoImport)
	}
}

func finishTypecheck(tb testing.TB, tc *TypeChecker, opts testutil.TypecheckOpts, checkErr error) {
	tb.Helper()
	if opts.ExpectError != "" {
		if checkErr == nil {
			tb.Fatalf("expected typecheck error containing %q, got nil", opts.ExpectError)
		}
		if !strings.Contains(checkErr.Error(), opts.ExpectError) {
			tb.Fatalf("typecheck error %q does not contain %q", checkErr.Error(), opts.ExpectError)
		}
		return
	}
	if checkErr != nil {
		tb.Fatalf("typecheck: %v", checkErr)
	}
	if opts.SkipUnlessGoImport != "" && !tc.GoImportPackageLoaded(opts.SkipUnlessGoImport) {
		tb.Skipf("%s not loaded (go/packages or workspace)", opts.SkipUnlessGoImport)
	}
	if opts.SkipUnlessDotImport && !tc.HasDotImportPackages() {
		tb.Skip("dot-import packages not loaded (go/packages or workspace)")
	}
}

func runCheckTypes(tb testing.TB, log *logrus.Logger, nodes []ast.Node, opts testutil.TypecheckOpts) (*TypeChecker, error) {
	tb.Helper()
	if opts.ModuleProviders {
		tb.Fatal("ModuleProviders is not supported in typechecker.Typecheck; use transformergo.MustCompileGo")
	}
	tc := New(log, false)
	applyTypecheckOpts(tb, tc, opts)
	return tc, tc.CheckTypes(nodes)
}

// Typecheck parses and typechecks src. Returns typecheck/module errors without fataling.
func Typecheck(tb testing.TB, src string, opts testutil.TypecheckOpts) (*TypeChecker, []ast.Node, error) {
	tb.Helper()
	log := opts.Logger
	if log == nil {
		log = testutil.TestLogger(tb, nil)
	}
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		tb.Fatalf("parse: %v", err)
	}
	tc, checkErr := runCheckTypes(tb, log, nodes, opts)
	if opts.ExpectError != "" {
		finishTypecheck(tb, tc, opts, checkErr)
		return tc, nodes, checkErr
	}
	if checkErr != nil {
		return tc, nodes, checkErr
	}
	finishTypecheck(tb, tc, opts, checkErr)
	return tc, nodes, nil
}

// MustTypecheck parses and typechecks src, fatals on unexpected errors.
func MustTypecheck(tb testing.TB, src string, opts testutil.TypecheckOpts) (*TypeChecker, []ast.Node) {
	tb.Helper()
	tc, nodes, err := Typecheck(tb, src, opts)
	if opts.ExpectError != "" {
		return tc, nodes
	}
	if err != nil {
		tb.Fatalf("typecheck: %v", err)
	}
	return tc, nodes
}

// MustTypecheckMerged merges paths into one package, then typechecks.
func MustTypecheckMerged(tb testing.TB, paths []string, opts testutil.TypecheckOpts) (*TypeChecker, []ast.Node) {
	tb.Helper()
	log := opts.Logger
	if log == nil {
		log = testutil.TestLogger(tb, nil)
	}
	merged, _, err := forstpkg.ParseAndMergePackage(log, paths)
	if err != nil {
		tb.Fatalf("merge package: %v", err)
	}
	tc, checkErr := runCheckTypes(tb, log, merged, opts)
	finishTypecheck(tb, tc, opts, checkErr)
	return tc, merged
}

// MustTypecheckMixedPackage typechecks src in a mixed Go/Forst module fixture.
func MustTypecheckMixedPackage(tb testing.TB, root, importPath, src string) (*TypeChecker, []ast.Node) {
	tb.Helper()
	goload.ClearLoadCacheForTest()
	tc, nodes := MustTypecheck(tb, src, testutil.TypecheckOpts{
		FileID:              "mixed/main.ft",
		GoWorkspaceDir:      root,
		SamePackageGoImport: importPath,
	})
	if !tc.SamePackageGoLoaded() {
		tb.Skip("same-package Go not loaded (go/packages or environment)")
	}
	return tc, nodes
}

// NewTypeCheckerForBench returns a silent TypeChecker for benchmarks.
func NewTypeCheckerForBench(b *testing.B) *TypeChecker {
	b.Helper()
	log := logrus.New()
	log.SetOutput(nil)
	log.SetLevel(logrus.PanicLevel)
	return New(log, false)
}

func moduleRootFromWD(tb testing.TB) string {
	tb.Helper()
	return testutil.ModuleRoot(tb)
}

func moduleRootForProvidersTest(tb testing.TB) string {
	tb.Helper()
	return testutil.ModuleRoot(tb)
}

func moduleRootForResultTests(tb testing.TB) string {
	tb.Helper()
	return testutil.ModuleRoot(tb)
}

func parseAndCheck(tb testing.TB, src string) (*TypeChecker, error) {
	tb.Helper()
	tc, _, err := Typecheck(tb, src, testutil.TypecheckOpts{UseModuleRoot: true})
	return tc, err
}
