package devserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

// RuntimeRunDeps injects compile/run for tests.
type RuntimeRunDeps struct {
	NewCompiler         func(compiler.Args, *logrus.Logger) *compiler.Compiler
	CreateOutput        func(main, nodert, invoke string, extra map[string]string, extraImports map[string]string, boundary string) (string, error)
	RunProgram          func(outputPath, boundaryRoot string) error
	StartProgram        func(outputPath, boundaryRoot string) (*runningChild, error)
	NewHostOrchestrator func(log *logrus.Logger, boundaryRoot string, cfg *ftconfig.Config) *HostOrchestrator
}

var defaultRuntimeRunDeps = RuntimeRunDeps{
	NewCompiler:         compiler.New,
	CreateOutput:        compiler.CreateTempOutputFiles,
	RunProgram:          compiler.RunGoProgram,
	StartProgram:        defaultStartProgram,
	NewHostOrchestrator: NewHostOrchestrator,
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
	hostOrch, err := startHostOrchestrator(log, boundaryRoot, cfg, deps)
	if err != nil {
		return err
	}
	if hostOrch != nil {
		defer shutdownHostOrchestrator(log, hostOrch)
	}
	outputPath, err := compileRuntimeOutput(log, boundaryRoot, entryPath, cfg, deps)
	if err != nil {
		return err
	}
	return deps.RunProgram(outputPath, boundaryRoot)
}

// WatchRuntimeDev compiles and runs the entry, then watches .ft sources for changes.
func WatchRuntimeDev(log *logrus.Logger, boundaryRoot, entryPath string, cfg *ftconfig.Config, deps RuntimeRunDeps) error {
	deps = fillRuntimeRunDeps(deps)
	hostOrch, err := startHostOrchestrator(log, boundaryRoot, cfg, deps)
	if err != nil {
		return err
	}
	if hostOrch != nil {
		defer shutdownHostOrchestrator(log, hostOrch)
	}
	autoRestart := cfg == nil || cfg.Dev.AutoRestart
	invokeAddr := InvokeListenAddr(cfg)
	healthURL := InvokeBaseURL(cfg) + "/health"

	var (
		child      *runningChild
		generation uint64
	)
	runReload := func() {
		gen := atomic.AddUint64(&generation, 1)
		child = performDevReload(reloadParams{
			log:          log,
			boundaryRoot: boundaryRoot,
			entryPath:    entryPath,
			cfg:          cfg,
			deps:         deps,
			autoRestart:  autoRestart,
			invokeAddr:   invokeAddr,
			healthURL:    healthURL,
			gen:          gen,
			child:        child,
		})
	}

	coalescer := newReloadCoalescer(runReload)
	coalescer.runSync()
	return WatchPackageRoot(log, boundaryRoot, cfg, defaultWatchDebounce, func() {
		if coalescer.schedule() {
			log.Info("File changed, recompiling...")
		} else {
			log.Debug("Reload in progress; coalesced file change into one follow-up recompile")
		}
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
	if deps.NewHostOrchestrator == nil {
		deps.NewHostOrchestrator = defaultRuntimeRunDeps.NewHostOrchestrator
	}
	return deps
}

func startHostOrchestrator(log *logrus.Logger, boundaryRoot string, cfg *ftconfig.Config, deps RuntimeRunDeps) (*HostOrchestrator, error) {
	if cfg == nil || !cfg.Node.HostMode {
		return nil, nil
	}
	hostOrch := deps.NewHostOrchestrator(log, boundaryRoot, cfg)
	if err := hostOrch.EnsureRunning(); err != nil {
		return nil, fmt.Errorf("node host: %w", err)
	}
	return hostOrch, nil
}

func shutdownHostOrchestrator(log *logrus.Logger, hostOrch *HostOrchestrator) {
	if hostOrch == nil {
		return
	}
	if err := hostOrch.Shutdown(); err != nil && log != nil {
		log.Warnf("node host shutdown: %v", err)
	}
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

type reloadParams struct {
	log          *logrus.Logger
	boundaryRoot string
	entryPath    string
	cfg          *ftconfig.Config
	deps         RuntimeRunDeps
	autoRestart  bool
	invokeAddr   string
	healthURL    string
	gen          uint64
	child        *runningChild
}

func performDevReload(p reloadParams) *runningChild {
	log := p.log
	boundaryRoot := p.boundaryRoot
	gen := p.gen
	child := p.child

	log.Info("Pausing invoke server for recompile...")
	if err := MarkReloading(boundaryRoot, true, gen); err != nil {
		log.Warnf("reload marker: %v", err)
	}
	if err := RemoveInvokeReady(boundaryRoot); err != nil {
		log.Warnf("remove invoke.ready: %v", err)
	}

	outputPath, err := compileRuntimeOutput(log, boundaryRoot, p.entryPath, p.cfg, p.deps)
	if err != nil {
		log.Error(err)
		log.Warn("Invoke server is down until compilation succeeds on next save")
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		return child
	}
	if !p.autoRestart {
		log.Info("Compile succeeded (dev.autoRestart is false; not restarting process)")
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		return child
	}

	if child != nil {
		child.stopForReload(log)
		child = nil
	}
	if err := WaitPortFree(p.invokeAddr, defaultPortFreeTimeout); err != nil {
		log.Error(err)
	}

	next, err := p.deps.StartProgram(outputPath, boundaryRoot)
	if err != nil {
		log.Error(err)
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		return child
	}

	if err := invokeReadyWaiter(boundaryRoot, p.healthURL, next.exited, defaultInvokeReadyWait); err != nil {
		log.Errorf("reload failed: invoke server did not become ready: %v", err)
		_ = next.stop()
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		return child
	}

	if err := MarkReloading(boundaryRoot, false, gen); err != nil {
		log.Warnf("clear reload marker: %v", err)
	}
	log.Infof("Invoke server ready after reload (generation=%d)", gen)
	monitorChildExit(log, next)
	return next
}
