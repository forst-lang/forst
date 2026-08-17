package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"forst/internal/forstpkg"
	"forst/internal/programbuild"
)

// BuildNativeProgram transpiles, links, and writes manifest.json under outputDir.
func (c *Compiler) BuildNativeProgram(outputDir, goos, goarch string) error {
	if err := programbuild.ValidateOutputPath(outputDir); err != nil {
		return err
	}
	if !c.embedInvokeEnabled() {
		return fmt.Errorf("forst build requires server.embedded: true in ftconfig.json (or enable embedded invoke in config)")
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	sandboxMain, boundaryRoot, extraPkgs, err := c.compileProgramSandbox()
	if err != nil {
		return err
	}

	absOut, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("resolve output dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(absOut, programbuild.BinDir), 0o755); err != nil {
		return fmt.Errorf("create output bin dir: %w", err)
	}

	binName, err := programbuild.BinaryFileName(c.Args.FilePath, goos)
	if err != nil {
		return err
	}
	binRel := filepath.Join(programbuild.BinDir, binName)
	binPath := filepath.Join(absOut, binRel)
	if err := BuildGoProgramInSandboxWithTarget(sandboxMain, binPath, boundaryRoot, goos, goarch); err != nil {
		return err
	}

	entryRel, err := relativeToBoundary(boundaryRoot, c.Args.FilePath)
	if err != nil {
		entryRel = c.Args.FilePath
	}

	manifest := programbuild.ProgramManifest{
		SchemaVersion:     programbuild.SchemaVersion,
		Kind:              programbuild.KindProgram,
		CompilerVersion:   CompilerVersion(),
		ContractVersion:   programbuild.ContractVersion,
		Entry:             entryRel,
		BoundaryRoot:      boundaryRoot,
		GOOS:              goos,
		GOARCH:            goarch,
		EmbeddedInvoke:    true,
		HostMode:          c.nodeHostModeEnabled(),
		SkipNodeHostDefault: false,
		Packages:          manifestPackages(c, extraPkgs),
		Binary:            filepath.ToSlash(binRel),
		BuiltAt:           time.Now().UTC().Format(time.RFC3339),
	}
	if err := programbuild.Write(absOut, manifest); err != nil {
		return err
	}
	c.log.Infof("Built program binary at %s", binPath)
	return nil
}

func (c *Compiler) compileProgramSandbox() (sandboxMain, boundaryRoot string, extraPkgs map[string]string, err error) {
	mainCode, nodeRuntime, invokeServer, extraPkgs, extraImports, err := c.CompileWithNodeRuntime()
	if err != nil {
		return "", "", nil, err
	}
	boundaryRoot = RunBoundaryRoot(c.Args)
	sandboxMain, err = CreateTempOutputFiles(mainCode, nodeRuntime, invokeServer, extraPkgs, extraImports, boundaryRoot)
	if err != nil {
		return "", "", nil, fmt.Errorf("prepare build sandbox: %w", err)
	}
	return sandboxMain, boundaryRoot, extraPkgs, nil
}

func manifestPackages(c *Compiler, extra map[string]string) []string {
	names := make(map[string]struct{})
	if c.Args.FilePath != "" {
		if nodes, err := c.loadInputNodesForCompile(); err == nil {
			pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
			if pkg != "" {
				names[pkg] = struct{}{}
			}
		}
	}
	for pkg := range extra {
		if pkg != "" {
			names[pkg] = struct{}{}
		}
	}
	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func relativeToBoundary(boundaryRoot, path string) (string, error) {
	if boundaryRoot == "" || path == "" {
		return path, fmt.Errorf("empty path")
	}
	absBoundary, err := filepath.Abs(boundaryRoot)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absBoundary, absPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}
