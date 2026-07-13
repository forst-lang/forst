package goload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvForstGOModRoot = "FORST_GOMOD_ROOT"

var (
	modRootExecutable     = os.Executable
	modRootGetwd          = os.Getwd
	modRootGetenv         = os.Getenv
	forstCompilerRootHook func() string
)

// SetForstCompilerModuleRootHookForTest overrides ForstCompilerModuleRoot in tests.
// Returns a restore function.
func SetForstCompilerModuleRootHookForTest(fn func() string) func() {
	prev := forstCompilerRootHook
	forstCompilerRootHook = fn
	return func() { forstCompilerRootHook = prev }
}

// IsForstCompilerModule reports whether dir is the Forst compiler Go module (contains cmd/forst).
func IsForstCompilerModule(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, "cmd", "forst"))
	return err == nil && st.IsDir()
}

// ForstCompilerModuleRoot returns the filesystem root of the Forst compiler Go module
// (module forst). Used when generated companion Go imports forst/nodert or forst/internal/*.
func ForstCompilerModuleRoot() string {
	if forstCompilerRootHook != nil {
		return forstCompilerRootHook()
	}
	if root := strings.TrimSpace(modRootGetenv(EnvForstGOModRoot)); root != "" {
		if IsForstCompilerModule(root) {
			return filepath.Clean(root)
		}
	}
	if exe, err := modRootExecutable(); err == nil {
		if resolved, symErr := filepath.EvalSymlinks(exe); symErr == nil {
			exe = resolved
		}
		if root := findForstCompilerModuleFrom(exe); root != "" {
			return root
		}
		if root := forstCompilerModuleAdjacentToExecutable(exe); root != "" {
			return root
		}
	}
	if wd, err := modRootGetwd(); err == nil {
		if root := findForstCompilerModuleFrom(wd); root != "" {
			return root
		}
	}
	return ""
}

func forstCompilerModuleAdjacentToExecutable(executable string) string {
	dir, err := filepath.Abs(filepath.Dir(executable))
	if err != nil {
		return ""
	}
	for _, candidate := range []string{
		filepath.Join(dir, "module"),
		filepath.Join(filepath.Dir(dir), "module"),
	} {
		if IsForstCompilerModule(candidate) {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func findForstCompilerModuleFrom(start string) string {
	startDir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		startDir = filepath.Dir(start)
	}
	dir := filepath.Clean(startDir)
	for {
		if IsForstCompilerModule(dir) {
			return dir
		}
		nested := filepath.Join(dir, "forst")
		if IsForstCompilerModule(nested) {
			return nested
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// ModuleRootWithGoMod walks upward from start until a directory containing go.mod
// is found. Unlike FindModuleRoot, this returns an error when no go.mod exists.
func ModuleRootWithGoMod(start string) (string, error) {
	startDir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		startDir = filepath.Dir(start)
	}
	startDir = filepath.Clean(startDir)
	dir := startDir
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod in %s or ancestor directories", startDir)
		}
		dir = parent
	}
}

// FindModuleRoot walks upward from start (file or directory) until a directory
// containing go.mod is found. If none is found, it returns the directory that
// contained start (the starting folder for a directory, or the file's parent).
//
// This is used as go/packages Config.Dir so Forst imports resolve against the
// same module as the surrounding Go project (e.g. sidecar / monorepo roots).
const forstGoModDir = ".forst-gomod"

// ForstGoModDir is the subdirectory name for Node-primary Go module shims.
const ForstGoModDir = forstGoModDir

// IsForstGoModShim reports whether moduleRoot is a .forst-gomod shim (no user Go packages).
func IsForstGoModShim(moduleRoot string) bool {
	return filepath.Base(filepath.Clean(moduleRoot)) == forstGoModDir
}

func FindModuleRoot(start string) string {
	startDir := start
	if info, err := os.Stat(start); err == nil && !info.IsDir() {
		startDir = filepath.Dir(start)
	}
	startDir = filepath.Clean(startDir)
	if modRoot := moduleRootInDir(startDir); modRoot != "" {
		return modRoot
	}
	dir := startDir
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			return startDir
		}
		dir = parent
		if modRoot := moduleRootInDir(dir); modRoot != "" {
			return modRoot
		}
	}
}

func moduleRootInDir(dir string) string {
	if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
		return dir
	}
	forstMod := filepath.Join(dir, forstGoModDir)
	if st, err := os.Stat(filepath.Join(forstMod, "go.mod")); err == nil && !st.IsDir() {
		return forstMod
	}
	return ""
}

// DirHasGoMod reports whether dir contains a go.mod file.
func DirHasGoMod(dir string) bool {
	if dir == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil && !st.IsDir()
}

// ModuleRootHasGoMod reports whether FindModuleRoot(start) resolved to a directory with go.mod.
func ModuleRootHasGoMod(start string) bool {
	return DirHasGoMod(FindModuleRoot(start))
}

// MissingGoModuleSetupHint returns setup guidance when boundaryRoot has no Go module.
func MissingGoModuleSetupHint(boundaryRoot string) string {
	if boundaryRoot == "" || ModuleRootHasGoMod(boundaryRoot) {
		return ""
	}
	return "create .forst-gomod/go.mod at " + filepath.Clean(boundaryRoot) +
		" with require for Go imports and replace forst => ... (see docs/interop/node/call-forst.mdx)"
}

// GoImportTypesNotLoadedMsg formats the go-import diagnostic for unloaded Go package types.
func GoImportTypesNotLoadedMsg(pkgName, importPath, workspaceDir, boundaryRoot string) string {
	msg := fmt.Sprintf("Go package %q (%s) types not loaded; check go.mod workspace and go tooling", pkgName, importPath)
	hintRoot := boundaryRoot
	if hintRoot == "" {
		hintRoot = workspaceDir
	}
	if hint := MissingGoModuleSetupHint(hintRoot); hint != "" {
		msg += "; " + hint
	}
	return msg
}
