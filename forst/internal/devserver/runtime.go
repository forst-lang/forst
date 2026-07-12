package devserver

import (
	"fmt"
	"os"
	"path/filepath"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

// RuntimeRunDeps injects compile/run for tests.
type RuntimeRunDeps struct {
	NewCompiler  func(compiler.Args, *logrus.Logger) *compiler.Compiler
	CreateOutput func(main, nodert, invoke string, extra map[string]string, extraImports map[string]string, boundary string) (string, error)
	RunProgram   func(outputPath, boundaryRoot string) error
}

var defaultRuntimeRunDeps = RuntimeRunDeps{
	NewCompiler:  compiler.New,
	CreateOutput: compiler.CreateTempOutputFiles,
	RunProgram:   compiler.RunGoProgram,
}

// ResolveEntry picks the .ft entry for runtime dev.
// Priority: cliEntry > cfg.Dev.Entry > forst/main.ft > main.ft
func ResolveEntry(boundaryRoot string, cfg *ftconfig.Config, cliEntry string) (string, error) {
	boundaryRoot = filepath.Clean(boundaryRoot)
	candidates := []string{}
	if cliEntry != "" {
		candidates = append(candidates, cliEntry)
	}
	if cfg != nil && cfg.Dev.Entry != "" {
		candidates = append(candidates, cfg.Dev.Entry)
	}
	candidates = append(candidates, "forst/main.ft", "main.ft")

	var tried []string
	for _, rel := range candidates {
		abs := rel
		if !filepath.IsAbs(rel) {
			abs = filepath.Join(boundaryRoot, rel)
		}
		abs = filepath.Clean(abs)
		tried = append(tried, abs)
		if st, err := os.Stat(abs); err == nil && !st.IsDir() {
			return abs, nil
		}
	}
	return "", fmt.Errorf(
		"runtime dev: no entry .ft found under %q (tried: %v); set dev.entry in ftconfig.json or pass -entry",
		boundaryRoot, tried,
	)
}

// RunRuntimeDev compiles and runs the entry binary (same pipeline as forst run).
func RunRuntimeDev(log *logrus.Logger, boundaryRoot, entryPath string, cfg *ftconfig.Config, deps RuntimeRunDeps) error {
	if deps.NewCompiler == nil {
		deps = defaultRuntimeRunDeps
	}
	args := compiler.Args{
		Command:            "run",
		FilePath:           entryPath,
		PackageRoot:        boundaryRoot,
		ExportStructFields: cfg != nil && cfg.Compiler.ExportStructFields,
		LogLevel:           "info",
	}
	if cfg != nil && cfg.Dev.LogLevel != "" {
		args.LogLevel = cfg.Dev.LogLevel
	}
	comp := deps.NewCompiler(args, log)
	mainCode, nodeRuntime, invokeCode, extraPkgs, extraImports, err := comp.CompileWithNodeRuntime()
	if err != nil {
		return err
	}
	outputPath, err := deps.CreateOutput(mainCode, nodeRuntime, invokeCode, extraPkgs, extraImports, boundaryRoot)
	if err != nil {
		return err
	}
	return deps.RunProgram(outputPath, boundaryRoot)
}
