package layout

import (
	"fmt"
	"path/filepath"
)

const (
	SuffixGen        = ".gen.go"
	FileInvokeServer = "forst_invoke_server.gen.go"
	FileNodeRuntime  = "forst_node_runtime.gen.go"
	FileTestWrapper  = "forst_gen_test.go"
	FileLibShim      = "forst_lib.gen.go"
)

// Root is the ftconfig boundary root (absolute).
type Root struct {
	Boundary string
}

// NewRoot builds a Root anchored at the ftconfig boundary directory; every
// generated-file path helper (.forst/run, .forst/exec, .forst/gen) is
// derived from this one boundary.
func NewRoot(boundary string) Root {
	return Root{Boundary: filepath.Clean(boundary)}
}

func (r Root) dotForst() string {
	return filepath.Join(r.Boundary, ".forst")
}

// RunSession returns paths for a forst run / dev runtime sandbox.
func (r Root) RunSession(sessionID string) SessionPaths {
	dir := filepath.Join(r.dotForst(), "run", sessionID)
	return SessionPaths{
		Dir:          dir,
		GoMod:        filepath.Join(dir, "go.mod"),
		HostMain:     filepath.Join(dir, "cmd", "host", "main"+SuffixGen),
		InvokeServer: filepath.Join(dir, "internal", "invoke", FileInvokeServer),
		NodeRuntime:  filepath.Join(dir, "internal", "nodert", FileNodeRuntime),
	}
}

// PackageGo returns the transpiled package file under the run session.
func (r Root) PackageGo(sessionID, importPathSuffix, pkgName string) string {
	if importPathSuffix == "" {
		importPathSuffix = pkgName
	}
	return filepath.Join(r.dotForst(), "run", sessionID, filepath.FromSlash(importPathSuffix), pkgName+SuffixGen)
}

// ExecModule returns paths for dev executor temp modules.
func (r Root) ExecModule(gen uint64, forstPkg, fn string) ExecPaths {
	dir := filepath.Join(r.dotForst(), "exec", fmt.Sprintf("%d", gen), forstPkg, fn)
	return ExecPaths{
		Dir:    dir,
		GoMod:  filepath.Join(dir, "go.mod"),
		Main:   filepath.Join(dir, "main"+SuffixGen),
		ExecGo: filepath.Join(dir, "forstexec", "forstexec"+SuffixGen),
	}
}

// TestRun returns paths for an ephemeral forst test session.
func (r Root) TestRun(runID, relPkg string) TestPaths {
	base := filepath.Join(r.dotForst(), "gen", "test", runID)
	return TestPaths{
		RunDir:   base,
		ModDir:   filepath.Join(base, "mod"),
		GoMod:    filepath.Join(base, "mod", "go.mod"),
		TestFile: filepath.Join(base, "test", relPkg, FileTestWrapper),
		LibDir:   filepath.Join(base, "lib"),
	}
}

// LibShim returns the session-scoped lib shim path for a package import suffix.
func (r Root) LibShim(runID, importPathSuffix string) string {
	return filepath.Join(r.dotForst(), "gen", "test", runID, "lib", filepath.FromSlash(importPathSuffix), FileLibShim)
}

// GoWork returns the auto-managed workspace path (Mode B fallback).
func (r Root) GoWork() string {
	return filepath.Join(r.dotForst(), "go.work")
}

// SessionPaths are the generated file locations for one forst run/dev
// runtime sandbox, as returned by Root.RunSession.
type SessionPaths struct {
	Dir, GoMod, HostMain, InvokeServer, NodeRuntime string
}

// ExecPaths are the generated file locations for one dev-executor temp
// module, as returned by Root.ExecModule.
type ExecPaths struct {
	Dir, GoMod, Main, ExecGo string
}

// TestPaths are the generated file locations for one ephemeral forst test
// run, as returned by Root.TestRun.
type TestPaths struct {
	RunDir, ModDir, GoMod, TestFile, LibDir string
}
