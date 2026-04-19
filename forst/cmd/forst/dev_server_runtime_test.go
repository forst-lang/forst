package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/discovery"

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
	cfg := loadAndValidateConfig(configPath, logger, "9090", &overrideLevel)
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

	cfg := loadAndValidateConfig(configPath, logger, "", nil)
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

		_ = loadAndValidateConfig(configPath, logger, "", nil)
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
	_ = loadAndValidateConfig(configPath, logger, "", nil)
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
		"HTTP server listening on port 8080",
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
		StartDevServer("8080", logger, "/path/that/does/not/exist/ftconfig.json", ".", &level)
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
	err := StartDevServer("invalid-port", logger, "", t.TempDir(), &level)
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
	if err := StartDevServer("8080", logger, "", t.TempDir(), &level); err != nil {
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
