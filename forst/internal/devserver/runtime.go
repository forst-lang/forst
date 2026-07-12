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
	StartProgram func(outputPath, boundaryRoot string) (*runningChild, error)
}

var defaultRuntimeRunDeps = RuntimeRunDeps{
	NewCompiler:  compiler.New,
	CreateOutput: compiler.CreateTempOutputFiles,
	RunProgram:   compiler.RunGoProgram,
	StartProgram: defaultStartProgram,
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
	deps = fillRuntimeRunDeps(deps)
	outputPath, err := compileRuntimeOutput(log, boundaryRoot, entryPath, cfg, deps)
	if err != nil {
		return err
	}
	return deps.RunProgram(outputPath, boundaryRoot)
}

// WatchRuntimeDev compiles and runs the entry, then watches .ft sources for changes.
func WatchRuntimeDev(log *logrus.Logger, boundaryRoot, entryPath string, cfg *ftconfig.Config, deps RuntimeRunDeps) error {
	deps = fillRuntimeRunDeps(deps)
	autoRestart := cfg == nil || cfg.Dev.AutoRestart

	var child *runningChild
	reload := func() {
		outputPath, err := compileRuntimeOutput(log, boundaryRoot, entryPath, cfg, deps)
		if err != nil {
			log.Error(err)
			if child != nil {
				log.Warn("Keeping previous build running because of compile errors")
			} else {
				log.Warn("Not running program because of errors during compilation")
			}
			return
		}
		if !autoRestart {
			log.Info("Compile succeeded (dev.autoRestart is false; not restarting process)")
			return
		}
		if child != nil {
			if stopErr := child.stop(); stopErr != nil {
				log.Debugf("Previous process exit: %v", stopErr)
			}
			child = nil
		}
		next, err := deps.StartProgram(outputPath, boundaryRoot)
		if err != nil {
			log.Error(err)
			return
		}
		child = next
	}

	reload()
	return WatchPackageRoot(log, boundaryRoot, cfg, defaultWatchDebounce, func() {
		log.Info("File changed, recompiling...")
		reload()
	})
}

func fillRuntimeRunDeps(deps RuntimeRunDeps) RuntimeRunDeps {
	if deps.NewCompiler == nil {
		deps.NewCompiler = defaultRuntimeRunDeps.NewCompiler
	}
	if deps.CreateOutput == nil {
		deps.CreateOutput = defaultRuntimeRunDeps.CreateOutput
	}
	if deps.RunProgram == nil {
		deps.RunProgram = defaultRuntimeRunDeps.RunProgram
	}
	if deps.StartProgram == nil {
		deps.StartProgram = defaultRuntimeRunDeps.StartProgram
	}
	return deps
}

func compileRuntimeOutput(log *logrus.Logger, boundaryRoot, entryPath string, cfg *ftconfig.Config, deps RuntimeRunDeps) (string, error) {
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
		return "", err
	}
	return deps.CreateOutput(mainCode, nodeRuntime, invokeCode, extraPkgs, extraImports, boundaryRoot)
}
