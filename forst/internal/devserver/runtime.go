package devserver

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"forst/internal/compiler"
	"forst/internal/devcompile"
	"forst/internal/ftconfig"
	"forst/internal/invokeserver"
	"forst/internal/modulecheck"

	"github.com/sirupsen/logrus"
)

type devShutdownState struct {
	log          *logrus.Logger
	boundaryRoot string
	hostOrch     *HostOrchestrator
	child        *runningChild
}

// performDevShutdown stops the go child, node host, and dev markers on exit.
func performDevShutdown(s devShutdownState) {
	if s.child != nil {
		s.child.intentionalStop.Store(true)
		_ = s.child.stop()
		go drainChildExit(s.child)
	}
	_ = ClearGoChildPID(s.boundaryRoot)
	_ = RemoveInvokeReady(s.boundaryRoot)
	shutdownHostOrchestrator(s.log, s.hostOrch)
}

// RuntimeRunDeps injects compile/run for tests.
type RuntimeRunDeps struct {
	NewCompiler         func(compiler.Args, *logrus.Logger) *compiler.Compiler
	CreateOutput        func(main, nodert, invoke string, extra map[string]string, extraImports map[string]string, boundary string) (string, error)
	// CreateOutputForReload overrides sandbox write during reload profiling (tests); nil uses CreateDevReloadOutputFiles.
	CreateOutputForReload func(main, nodert, invoke string, extra map[string]string, extraImports map[string]string, boundary string, sandbox *compiler.CompileSandboxTiming) (string, error)
	BuildProgram        func(mainGoPath, binPath, boundaryRoot string) error
	StartProgram        func(outputPath, boundaryRoot string) (*runningChild, error)
	RunProgram          func(outputPath, boundaryRoot string) error
	NewHostOrchestrator func(log *logrus.Logger, boundaryRoot string, cfg *ftconfig.Config) *HostOrchestrator
	DevSession          *devcompile.Session
	ModTidyCache        *compiler.SandboxModCache
}

type devReloadState struct {
	session      *devcompile.Session
	modTidyCache *compiler.SandboxModCache
	lastChanged  string
}

var defaultRuntimeRunDeps = RuntimeRunDeps{
	NewCompiler:         compiler.New,
	CreateOutput:        compiler.CreateTempOutputFiles,
	RunProgram:          compiler.RunGoProgram,
	BuildProgram:        compiler.BuildGoProgramInSandbox,
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
	outputPath, err := compileRuntimeOutput(log, boundaryRoot, entryPath, cfg, deps, false)
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

	state := &devReloadState{
		session:      deps.DevSession,
		modTidyCache: deps.ModTidyCache,
	}
	if state.session == nil {
		state.session = devcompile.NewSession(boundaryRoot)
		deps.DevSession = state.session
	}
	if state.modTidyCache == nil {
		state.modTidyCache = &compiler.SandboxModCache{}
		deps.ModTidyCache = state.modTidyCache
	}

	ReapOrphanedGoChild(boundaryRoot, 0, log)

	var (
		child      *runningChild
		generation uint64
	)
	runReload := func(changedPath string) {
		if changedPath != "" {
			state.session.NoteChange(changedPath)
			state.lastChanged = changedPath
		}
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
			state:        state,
		})
	}

	coalescer := newReloadCoalescer(func() { runReload(state.lastChanged) })

	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		if log != nil {
			log.Info("Shutting down forst dev...")
		}
		performDevShutdown(devShutdownState{
			log:          log,
			boundaryRoot: boundaryRoot,
			hostOrch:     hostOrch,
			child:        child,
		})
		close(stopCh)
	}()

	watchDone := make(chan error, 1)
	go func() {
		watchDone <- WatchPackageRoot(log, boundaryRoot, cfg, defaultWatchDebounce, func(changedPath string) {
			state.lastChanged = changedPath
			if state.session != nil && changedPath != "" {
				state.session.NoteChange(changedPath)
			}
			if coalescer.schedule() {
				log.Info("File changed, recompiling...")
			} else {
				log.Debug("Reload in progress; coalesced file change into one follow-up recompile")
			}
		}, stopCh)
	}()

	coalescer.runSync()
	return <-watchDone
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
	if deps.BuildProgram == nil {
		deps.BuildProgram = defaultRuntimeRunDeps.BuildProgram
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

func compileRuntimeOutput(log *logrus.Logger, boundaryRoot, entryPath string, cfg *ftconfig.Config, deps RuntimeRunDeps, reloadProfile bool) (string, error) {
	devReload := reloadProfile
	args := compiler.Args{
		Command:            "run",
		FilePath:           entryPath,
		PackageRoot:        boundaryRoot,
		ExportStructFields: cfg != nil && cfg.Compiler.ExportStructFields,
		LogLevel:           "info",
		ReloadProfile:      reloadProfile,
		DevStableSandbox:   devReload,
		DevModTidyCache:    deps.ModTidyCache,
		DevSession:         deps.DevSession,
	}
	if cfg != nil && cfg.Dev.LogLevel != "" {
		args.LogLevel = cfg.Dev.LogLevel
	}
	comp := deps.NewCompiler(args, log)
	mainCode, nodeRuntime, invokeCode, extraPkgs, extraImports, err := comp.CompileWithNodeRuntime()
	if err != nil {
		return "", err
	}
	if devReload {
		var sandbox compiler.CompileSandboxTiming
		var outputPath string
		if deps.CreateOutputForReload != nil {
			outputPath, err = deps.CreateOutputForReload(mainCode, nodeRuntime, invokeCode, extraPkgs, extraImports, boundaryRoot, &sandbox)
		} else {
			outputPath, err = compiler.CreateDevReloadOutputFiles(mainCode, nodeRuntime, invokeCode, extraPkgs, extraImports, boundaryRoot, deps.ModTidyCache, &sandbox)
		}
		if err != nil {
			return "", err
		}
		binPath := compiler.DevBinPath(boundaryRoot)
		buildStart := time.Now()
		if deps.BuildProgram == nil {
			err = compiler.BuildGoProgramInSandbox(outputPath, binPath, boundaryRoot)
		} else {
			err = deps.BuildProgram(outputPath, binPath, boundaryRoot)
		}
		if err != nil {
			return "", err
		}
		timings := comp.TakeCompileTimings()
		timings.WriteSandboxMs = sandbox.WriteSandboxMs
		timings.GoModTidyMs = sandbox.GoModTidyMs
		timings.GoBuildMs = time.Since(buildStart).Milliseconds()
		comp.LogCompilePhaseTiming(timings)
		// Return main.go path; StartGoProgram (go run) reuses the build cache from BuildGoProgramInSandbox.
		return outputPath, nil
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
	state        *devReloadState
}

func performDevReload(p reloadParams) *runningChild {
	log := p.log
	boundaryRoot := p.boundaryRoot
	gen := p.gen
	child := p.child
	profile := reloadProfileEnabled(p.cfg)

	var rt *reloadTiming
	if profile {
		modulecheck.ResetPassCount()
		rt = newReloadTiming()
		rt.logStart(log, gen)
	}

	log.Info("Pausing invoke server for recompile...")
	markStart := time.Now()
	if err := MarkReloading(boundaryRoot, true, gen); err != nil {
		log.Warnf("reload marker: %v", err)
	}
	if err := RemoveInvokeReady(boundaryRoot); err != nil {
		log.Warnf("remove invoke.ready: %v", err)
	}
	if rt != nil {
		rt.recordMarkReloading(markStart)
	}

	compileStart := time.Now()
	outputPath, err := compileRuntimeOutput(log, boundaryRoot, p.entryPath, p.cfg, p.deps, profile)
	if rt != nil {
		rt.recordForstCompile(compileStart)
	}
	if err != nil {
		log.Error(err)
		log.Warn("Invoke server is down until compilation succeeds on next save")
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		if rt != nil {
			rt.logComplete(log, gen, false)
		}
		return child
	}
	if !p.autoRestart {
		log.Info("Compile succeeded (dev.autoRestart is false; not restarting process)")
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		if rt != nil {
			rt.logComplete(log, gen, true)
		}
		return child
	}

	if child != nil {
		currentPID := child.pid
		ReapOrphanedGoChild(boundaryRoot, currentPID, log)

		stopStart := time.Now()
		child.stopForReload(log, boundaryRoot)
		child = nil
		if rt != nil {
			rt.recordChildStop(stopStart)
		}
	} else {
		ReapOrphanedGoChild(boundaryRoot, 0, log)
	}

	portStart := time.Now()
	host, preferred, _ := net.SplitHostPort(p.invokeAddr)
	if host == "" {
		host = "127.0.0.1"
	}
	invokePort, err := FindNextFreeInvokePort(host, preferred)
	if rt != nil {
		rt.recordPortPick(portStart)
	}
	if err != nil {
		log.Error(err)
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		if rt != nil {
			rt.logComplete(log, gen, false)
		}
		return child
	}
	if invokePort != preferred {
		log.Info(formatInvokePortShiftLog(preferred, invokePort))
	}
	_ = os.Setenv(invokeserver.EnvInvokePort, invokePort)
	healthURL := InvokeBaseURLFromHostPort(host, invokePort) + "/health"

	goStart := time.Now()
	next, err := p.deps.StartProgram(outputPath, boundaryRoot)
	if rt != nil {
		rt.recordGoStart(goStart)
	}
	if err != nil {
		log.Error(err)
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		if rt != nil {
			rt.logComplete(log, gen, false)
		}
		return child
	}
	recordStartedChild(boundaryRoot, next)

	if err := MarkReloading(boundaryRoot, false, gen); err != nil {
		log.Warnf("clear reload marker before invoke ready: %v", err)
	}

	readyStart := time.Now()
	if err := invokeReadyWaiter(boundaryRoot, healthURL, next.exited, defaultInvokeReadyWait); err != nil {
		log.Errorf("reload failed: invoke server did not become ready: %v", err)
		_ = next.stop()
		if clearErr := MarkReloading(boundaryRoot, false, gen); clearErr != nil {
			log.Warnf("clear reload marker: %v", clearErr)
		}
		if rt != nil {
			rt.recordInvokeReady(readyStart)
			rt.logComplete(log, gen, false)
		}
		return child
	}
	if rt != nil {
		rt.recordInvokeReady(readyStart)
	}

	log.Infof("Invoke server ready after reload (generation=%d)", gen)
	if readyURL, readErr := ReadInvokeReadyURL(boundaryRoot); readErr == nil && readyURL != "" {
		log.Infof("Invoke server URL: %s", readyURL)
	}
	if rt != nil {
		rt.logComplete(log, gen, true)
	}
	monitorChildExit(log, next)
	return next
}
