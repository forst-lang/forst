package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"forst/internal/ftconfig"
	"forst/internal/forstpkg"
	"forst/internal/programbuild"
)

// BuildNativeProgram transpiles, links, and writes manifest.json under outputDir.
// Without server.embedded, it emits plain Go into the entry package and wraps `go build`.
func (c *Compiler) BuildNativeProgram(outputDir, goos, goarch string) error {
	if err := programbuild.ValidateOutputPath(outputDir); err != nil {
		return err
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if !c.embedInvokeEnabled() {
		return c.buildPlainGoPackage(outputDir, goos, goarch)
	}
	return c.buildEmbeddedInvokeProgram(outputDir, goos, goarch)
}

func (c *Compiler) buildPlainGoPackage(outputDir, goos, goarch string) error {
	if c.Args.OutputPath == "" {
		if out := c.defaultPackageGoOut(); out != "" {
			c.Args.OutputPath = out
		}
	}
	if c.Args.OutputPath == "" {
		return fmt.Errorf("forst build: cannot resolve package Go output path")
	}
	mainCode, _, _, _, _, err := c.CompileWithBridgeRuntime()
	if err != nil {
		return err
	}
	if dir := filepath.Dir(c.Args.OutputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create Go output directory: %w", err)
		}
	}
	if err := os.WriteFile(c.Args.OutputPath, []byte(mainCode), 0o644); err != nil {
		return fmt.Errorf("write Go sources: %w", err)
	}
	pkgDir := filepath.Dir(c.Args.OutputPath)
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

	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = pkgDir
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
	if goos != runtime.GOOS || goarch != runtime.GOARCH {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build in %s: %w", pkgDir, err)
	}

	boundaryRoot := RunBoundaryRoot(c.Args)
	entryRel, err := relativeToBoundary(boundaryRoot, c.Args.FilePath)
	if err != nil {
		entryRel = c.Args.FilePath
	}
	manifest := programbuild.ProgramManifest{
		SchemaVersion:    programbuild.SchemaVersion,
		Kind:             programbuild.KindProgram,
		CompilerVersion:  CompilerVersion(),
		ContractVersion:  programbuild.ContractVersion,
		Entry:            entryRel,
		BoundaryRoot:     boundaryRoot,
		GOOS:             goos,
		GOARCH:           goarch,
		EmbeddedInvoke:   false,
		Packages:         manifestPackages(c, nil),
		Binary:           filepath.ToSlash(binRel),
		BuiltAt:          time.Now().UTC().Format(time.RFC3339),
	}
	if err := programbuild.Write(absOut, manifest); err != nil {
		return err
	}
	c.log.Infof("Built package binary at %s", binPath)
	return nil
}

func (c *Compiler) buildEmbeddedInvokeProgram(outputDir, goos, goarch string) error {
	sandboxMain, boundaryRoot, bridgeRuntime, extraPkgs, err := c.compileProgramSandbox()
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

	needsBridge := bridgeRuntime != ""
	compiledModulesDir := ""
	legacyModuleFormat := ""
	if needsBridge {
		cfg, cfgErr := c.loadFtconfig()
		if cfgErr == nil && cfg != nil {
			if bridge, bridgeErr := ftconfig.EffectiveBridge(cfg); bridgeErr == nil {
				legacyModuleFormat = string(bridge.ModuleFormat)
				if bridge.ModuleFormat == ftconfig.LegacyModuleCompiled {
					if dir := strings.TrimSpace(cfg.Bridge.LegacyModules.Dir); dir != "" {
						compiledModulesDir = filepath.ToSlash(filepath.Clean(dir))
					} else {
						compiledModulesDir = bridge.OutDir
					}
				}
			}
		}
	}

	entryRel, err := relativeToBoundary(boundaryRoot, c.Args.FilePath)
	if err != nil {
		entryRel = c.Args.FilePath
	}

	manifest := programbuild.ProgramManifest{
		SchemaVersion:       programbuild.SchemaVersion,
		Kind:                programbuild.KindProgram,
		CompilerVersion:     CompilerVersion(),
		ContractVersion:     programbuild.ContractVersion,
		Entry:               entryRel,
		BoundaryRoot:        boundaryRoot,
		GOOS:                goos,
		GOARCH:              goarch,
		EmbeddedInvoke:      true,
		HostMode:            c.bridgeHostModeEnabled(),
		SkipNodeHostDefault: false,
		NeedsBridgeRuntime:  needsBridge,
		CompiledModulesDir:  compiledModulesDir,
		LegacyModuleFormat:  legacyModuleFormat,
		Packages:            manifestPackages(c, extraPkgs),
		Binary:              filepath.ToSlash(binRel),
		BuiltAt:             time.Now().UTC().Format(time.RFC3339),
	}
	if err := programbuild.Write(absOut, manifest); err != nil {
		return err
	}
	c.log.Infof("Built program binary at %s", binPath)
	return nil
}

func (c *Compiler) compileProgramSandbox() (sandboxMain, boundaryRoot, bridgeRuntime string, extraPkgs map[string]string, err error) {
	mainCode, bridgeRuntime, invokeServer, extraPkgs, extraImports, err := c.CompileWithBridgeRuntime()
	if err != nil {
		return "", "", "", nil, err
	}
	boundaryRoot = RunBoundaryRoot(c.Args)
	sandboxMain, err = CreateTempOutputFiles(mainCode, bridgeRuntime, invokeServer, extraPkgs, extraImports, boundaryRoot)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("prepare build sandbox: %w", err)
	}
	return sandboxMain, boundaryRoot, bridgeRuntime, extraPkgs, nil
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
