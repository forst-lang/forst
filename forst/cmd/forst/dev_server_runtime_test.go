package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"
	"forst/internal/discovery"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

func TestDevServer_Stop_nilServerNoop(t *testing.T) {
	s := &DevServer{}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestDevServer_Stop_nonNilServerClosePath(t *testing.T) {
	// A non-started http.Server still exercises the non-nil branch in Stop().
	s := &DevServer{server: &http.Server{}}
	if err := s.Stop(); err != nil {
		t.Fatalf("expected nil close error for empty server, got %v", err)
	}
}

func TestLoadAndValidateConfig_overridesPortAndLogLevel(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ftconfig.json")
	configJSON := `{
  "server": { "port": "3001" },
  "dev": { "logLevel": "info" }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	overrideLevel := "debug"
	cfg := loadAndValidateConfig(configPath, logger, "9090", &overrideLevel, dir, false)
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Server.Port != "9090" {
		t.Fatalf("expected overridden port 9090, got %q", cfg.Server.Port)
	}
	if cfg.Dev.LogLevel != "debug" {
		t.Fatalf("expected overridden log level debug, got %q", cfg.Dev.LogLevel)
	}
}

func TestLoadAndValidateConfig_setsLoggerLevelFromConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ftconfig.json")
	configJSON := `{
  "server": { "port": "8080" },
  "dev": { "logLevel": "trace" }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.ErrorLevel)

	cfg := loadAndValidateConfig(configPath, logger, "", nil, dir, false)
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Dev.LogLevel != "trace" {
		t.Fatalf("expected config log level trace, got %q", cfg.Dev.LogLevel)
	}
	if logger.GetLevel() != logrus.TraceLevel {
		t.Fatalf("expected logger level trace, got %s", logger.GetLevel())
	}
}

func TestLoadAndValidateConfig_exportStructFieldsCLI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ftconfig.json")
	configJSON := `{
  "server": { "port": "8080" },
  "compiler": { "exportStructFields": false },
  "dev": { "logLevel": "info" }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	cfg := loadAndValidateConfig(configPath, logger, "", nil, dir, true)
	if cfg == nil || !cfg.Compiler.ExportStructFields {
		t.Fatalf("expected CLI exportStructFields override, got %+v", cfg)
	}
}

func TestLoadAndValidateConfig_warnAndInvalidLogLevelBehavior(t *testing.T) {
	t.Run("warn from config", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, "ftconfig.json")
		configJSON := `{
  "server": { "port": "8080" },
  "dev": { "logLevel": "warn" }
}`
		if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
			t.Fatal(err)
		}
		logger := logrus.New()
		logger.SetOutput(io.Discard)
		logger.SetLevel(logrus.DebugLevel)

		_ = loadAndValidateConfig(configPath, logger, "", nil, dir, false)
		if logger.GetLevel() != logrus.WarnLevel {
			t.Fatalf("expected warn level, got %s", logger.GetLevel())
		}
	})
}

func TestLoadAndValidateConfig_helperProcess_invalidLogLevel(_ *testing.T) {
	if os.Getenv("FORST_DEVCFG_HELPER") != "1" {
		return
	}
	configPath := os.Getenv("FORST_DEVCFG_PATH")
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	_ = loadAndValidateConfig(configPath, logger, "", nil, ".", false)
}

func TestLoadAndValidateConfig_invalidLogLevelExits(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ftconfig.json")
	configJSON := `{
  "server": { "port": "8080" },
  "dev": { "logLevel": "not-a-level" }
}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestLoadAndValidateConfig_helperProcess_invalidLogLevel")
	cmd.Env = append(os.Environ(), "FORST_DEVCFG_HELPER=1", "FORST_DEVCFG_PATH="+configPath)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for invalid dev.logLevel config")
	}
}

func TestDevServer_Start_bindsLoopbackByDefault(t *testing.T) {
	s := testDevServer(t)
	if s.host != "127.0.0.1" {
		t.Fatalf("host = %q want 127.0.0.1", s.host)
	}
	s.port = "0"

	ln, err := net.Listen("tcp", s.listenAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	tcpAddr := ln.Addr().(*net.TCPAddr)
	if !tcpAddr.IP.IsLoopback() {
		t.Fatalf("bound to non-loopback %v", tcpAddr.IP)
	}
}

func TestDevServer_Start_respectsExplicitZeroHost(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	cfg := DefaultConfig()
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.ReadTimeout = 1
	cfg.Server.WriteTimeout = 1
	comp := compiler.New(cfg.ToCompilerArgs(), log)
	s := NewHTTPServer("0", comp, log, cfg, t.TempDir())
	if s.host != "0.0.0.0" {
		t.Fatalf("host = %q want 0.0.0.0", s.host)
	}

	ln, err := net.Listen("tcp", s.listenAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	tcpAddr := ln.Addr().(*net.TCPAddr)
	if tcpAddr.IP.IsLoopback() {
		t.Fatalf("expected non-loopback bind for 0.0.0.0, got %v", tcpAddr.IP)
	}
}

func TestDevServer_Start_invalidPortReturnsError_andInitializesTypesGenerator(t *testing.T) {
	s := testDevServer(t)
	s.port = "invalid-port"
	err := s.Start()
	if err == nil {
		t.Fatal("expected start error for invalid port")
	}
	if s.typesGenerator == nil {
		t.Fatal("expected types generator initialization before listen failure")
	}
}

func TestDevServer_logStartupInfo_includesEndpoints(t *testing.T) {
	s := testDevServer(t)
	buf := &bytes.Buffer{}
	s.log.SetOutput(buf)
	s.port = "8080"

	s.logStartupInfo()

	output := buf.String()
	for _, fragment := range []string{
		"HTTP server listening on 127.0.0.1:8080",
		"GET  /functions",
		"POST /invoke",
		"GET  /types",
		"GET  /health",
		"GET  /version",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("startup log missing %q in output:\n%s", fragment, output)
		}
	}
}

func TestStartDevServer_helperProcess(t *testing.T) {
	helperCase := os.Getenv("FORST_START_DEVSERVER_HELPER_CASE")
	if helperCase == "" {
		return
	}

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "info"

	switch helperCase {
	case "invalid-config-path":
		_ = StartDevServer(ftconfig.DefaultDevExecutorPort, logger, "/path/that/does/not/exist/ftconfig.json", ".", &level, false, "")
	default:
		t.Fatalf("unknown helper case: %s", helperCase)
	}
}

func TestStartDevServer_exitsOnConfigLoadFailure(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestStartDevServer_helperProcess")
	cmd.Env = append(os.Environ(), "FORST_START_DEVSERVER_HELPER_CASE=invalid-config-path")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected StartDevServer to exit non-zero on config load failure")
	}
}

func TestStartDevServer_returnsErrorOnServerStartFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "info"
	err := StartDevServer("invalid-port", logger, "", t.TempDir(), &level, false, "")
	if err == nil {
		t.Fatal("expected error when listen address is invalid")
	}
}

func TestStartDevServer_returnsNilWhenStartHookSucceeds(t *testing.T) {
	orig := devServerStartFn
	devServerStartFn = func(*DevServer) error { return nil }
	t.Cleanup(func() { devServerStartFn = orig })

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "info"
	if err := StartDevServer(ftconfig.DefaultDevExecutorPort, logger, "", t.TempDir(), &level, false, ""); err != nil {
		t.Fatalf("expected nil when start hook succeeds, got %v", err)
	}
}

func TestDevServer_discoverFunctions_successUpdatesCache(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	rootDir := t.TempDir()
	cfg := DefaultConfig()

	s := &DevServer{
		log:        log,
		discoverer: discovery.NewDiscoverer(rootDir, log, cfg),
		functions: map[string]map[string]discovery.FunctionInfo{
			"stale": {
				"Old": {Package: "stale", Name: "Old"},
			},
		},
	}

	if err := s.discoverFunctions(); err != nil {
		t.Fatalf("discoverFunctions success path returned error: %v", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.functions == nil {
		t.Fatal("expected functions map to be set")
	}
	if _, ok := s.functions["stale"]; ok {
		t.Fatalf("expected stale map to be replaced by fresh discovery result, got %+v", s.functions)
	}
}

func TestDevServer_Start_logsWarningWhenInitialDiscoveryFails(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	s := testDevServer(t)
	s.log = log
	s.port = "invalid-port"
	// nil config in discoverer forces initial discovery failure; Start should warn and continue to listen path.
	s.discoverer = discovery.NewDiscoverer(t.TempDir(), log, nil)

	err := s.Start()
	if err == nil {
		t.Fatal("expected listen error due to invalid port")
	}
	if s.typesGenerator == nil {
		t.Fatal("expected types generator to be initialized even when discovery fails")
	}
}

func TestDevServer_discoverFunctions_errorPropagates(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	s := &DevServer{
		log:        log,
		discoverer: discovery.NewDiscoverer(t.TempDir(), log, nil),
	}

	err := s.discoverFunctions()
	if err == nil {
		t.Fatal("expected discoverFunctions error")
	}
	if !strings.Contains(err.Error(), "failed to discover functions") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDevServer_refreshFunctions_syncsDevBackendForInvoke(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "in", "tictactoe")
	if _, err := os.Stat(root); err != nil {
		t.Skip("tictactoe example not available")
	}

	log := logrus.New()
	log.SetOutput(io.Discard)
	cfg := DefaultConfig()
	comp := compiler.New(cfg.ToCompilerArgs(), log)
	s := NewHTTPServer("8090", comp, log, cfg, root)

	if err := s.refreshFunctions(); err != nil {
		t.Fatalf("refreshFunctions: %v", err)
	}

	fns := s.devBackend.Functions()
	mainPkg, ok := fns["main"]
	if !ok {
		t.Fatalf("expected main package in dev backend, got %+v", fns)
	}
	if _, ok := mainPkg["NewGame"]; !ok {
		t.Fatalf("expected main.NewGame in dev backend, got %+v", mainPkg)
	}
}

func TestStartDevServer_runtimeProfile_callsRunRuntimeDev(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotEntry string
	origRun := runRuntimeDevFn
	runRuntimeDevFn = func(_ *logrus.Logger, boundaryRoot, entry string, _ *ftconfig.Config) error {
		if boundaryRoot != dir {
			t.Fatalf("boundaryRoot = %q want %q", boundaryRoot, dir)
		}
		gotEntry = entry
		return nil
	}
	t.Cleanup(func() { runRuntimeDevFn = origRun })

	origStart := devServerStartFn
	devServerStartFn = func(*DevServer) error {
		t.Fatal("executor HTTP server must not start in runtime profile")
		return nil
	}
	t.Cleanup(func() { devServerStartFn = origStart })

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "error"
	if err := StartDevServer("", logger, filepath.Join(dir, "ftconfig.json"), dir, &level, false, ""); err != nil {
		t.Fatal(err)
	}
	if gotEntry != mainPath {
		t.Fatalf("entry = %q want %q", gotEntry, mainPath)
	}
}

func TestStartDevServer_runtimeProfile_missingEntry(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "error"
	err := StartDevServer("", logger, filepath.Join(dir, "ftconfig.json"), dir, &level, false, "")
	if err == nil || !strings.Contains(err.Error(), "no entry .ft found") {
		t.Fatalf("want missing entry error, got %v", err)
	}
}

func TestStartDevServer_executorProfile_unchanged(t *testing.T) {
	origRun := runRuntimeDevFn
	runRuntimeDevFn = func(*logrus.Logger, string, string, *ftconfig.Config) error {
		t.Fatal("runtime dev must not run in executor profile")
		return nil
	}
	t.Cleanup(func() { runRuntimeDevFn = origRun })

	var executorStarted bool
	origStart := devServerStartFn
	devServerStartFn = func(*DevServer) error {
		executorStarted = true
		return nil
	}
	t.Cleanup(func() { devServerStartFn = origStart })

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "error"
	if err := StartDevServer("", logger, "", t.TempDir(), &level, false, ""); err != nil {
		t.Fatal(err)
	}
	if !executorStarted {
		t.Fatal("expected executor HTTP path")
	}
}

func TestStartDevServer_runtimeProfile_setsInvokePort(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true,"port":"6321"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origRun := runRuntimeDevFn
	runRuntimeDevFn = func(*logrus.Logger, string, string, *ftconfig.Config) error { return nil }
	t.Cleanup(func() { runRuntimeDevFn = origRun })

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "error"
	if err := StartDevServer("6391", logger, filepath.Join(dir, "ftconfig.json"), dir, &level, false, ""); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("FORST_INVOKE_PORT"); got != "6391" {
		t.Fatalf("FORST_INVOKE_PORT = %q want 6391", got)
	}
}

func TestStartDevServer_runtimeProfile_usesFtconfigPortWhenNoCliOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true,"port":"6321"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ft"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origRun := runRuntimeDevFn
	runRuntimeDevFn = func(*logrus.Logger, string, string, *ftconfig.Config) error { return nil }
	t.Cleanup(func() { runRuntimeDevFn = origRun })

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "error"
	if err := StartDevServer("", logger, filepath.Join(dir, "ftconfig.json"), dir, &level, false, ""); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("FORST_INVOKE_PORT"); got != "6321" {
		t.Fatalf("FORST_INVOKE_PORT = %q want 6321", got)
	}
}

func TestStartDevServer_explicitExecutorOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"server":{"embedded":true},"dev":{"profile":"executor"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	origRun := runRuntimeDevFn
	runRuntimeDevFn = func(*logrus.Logger, string, string, *ftconfig.Config) error {
		t.Fatal("runtime dev must not run when profile is executor")
		return nil
	}
	t.Cleanup(func() { runRuntimeDevFn = origRun })

	var executorStarted bool
	origStart := devServerStartFn
	devServerStartFn = func(*DevServer) error {
		executorStarted = true
		return nil
	}
	t.Cleanup(func() { devServerStartFn = origStart })

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	level := "error"
	if err := StartDevServer("", logger, filepath.Join(dir, "ftconfig.json"), dir, &level, false, ""); err != nil {
		t.Fatal(err)
	}
	if !executorStarted {
		t.Fatal("expected executor path with dev.profile executor override")
	}
}

func TestStartDevServer_discoversFtconfigFromRootDir(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "project")
	if err := os.MkdirAll(filepath.Join(root, "forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "forst", "main.ft"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(root, "ftconfig.json")
	if err := os.WriteFile(cfgPath, []byte(`{"server":{"embedded":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	otherCwd := filepath.Join(outer, "other")
	if err := os.MkdirAll(otherCwd, 0o755); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(otherCwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	var logBuf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuf)
	level := "error"

	origRun := runRuntimeDevFn
	runRuntimeDevFn = func(_ *logrus.Logger, _, _ string, cfg *ftconfig.Config) error {
		if !cfg.Server.Embedded {
			t.Fatal("expected embedded from ftconfig discovered via -root")
		}
		return nil
	}
	t.Cleanup(func() { runRuntimeDevFn = origRun })

	if err := StartDevServer("", logger, "", root, &level, false, ""); err != nil {
		t.Fatalf("StartDevServer: %v", err)
	}
	if !strings.Contains(logBuf.String(), "Loaded config from:") {
		t.Fatalf("expected loaded config log, got: %s", logBuf.String())
	}
}
