package testutil

import "github.com/sirupsen/logrus"

// TypecheckOpts configures typecheck harness helpers in typechecker and testpipeline packages.
type TypecheckOpts struct {
	FileID              string // default "test.ft"
	GoWorkspaceDir      string // explicit; use UseModuleRoot to set from cwd walk
	NodeBoundaryRoot    string // project root for TS imports; defaults to GoWorkspaceDir
	ForstFileDir        string // directory of the Forst source file for relative TS imports
	NodeImportPolicy    string // ftconfig node.importPolicy override ("explicit" or "implicit")
	UseModuleRoot       bool   // set GoWorkspaceDir from ModuleRoot(tb)
	SamePackageGoImport string // sets tc.SetSamePackageGoImportPath
	ModuleProviders     bool   // run modulecheck.CheckModuleProviders before CheckTypes
	SkipUnlessGoImport  string // tb.Skip if import local name not loaded
	SkipUnlessDotImport bool
	ExpectError         string // non-empty: require CheckTypes error containing substring
	Logger              *logrus.Logger
}

// CompileOpts configures compile harness helpers in testpipeline and transformergo tests.
type CompileOpts struct {
	TypecheckOpts
	ExportStructFields bool
	FormatGo           bool // run go/format.Source on output
}
