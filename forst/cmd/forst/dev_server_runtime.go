package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"forst/internal/compiler"
	"forst/internal/devserver"
	"forst/internal/ftconfig"

	logrus "github.com/sirupsen/logrus"
)

// listenAddr returns host:port for the dev HTTP listener.
func (s *DevServer) listenAddr() string {
	return s.host + ":" + s.port
}

// Start starts the HTTP server.
func (s *DevServer) Start() error {
	if err := s.refreshFunctions(); err != nil {
		s.log.Warnf("Failed to discover functions on startup: %v", err)
	}

	s.typesGenerator = NewTypeScriptGenerator(s.log)

	mux := http.NewServeMux()
	s.invoke.RegisterRoutes(mux)
	mux.HandleFunc("/types", s.handleTypes)

	readTimeout := time.Duration(s.config.Server.ReadTimeout) * time.Second
	writeTimeout := time.Duration(s.config.Server.WriteTimeout) * time.Second

	s.server = &http.Server{
		Addr:         s.listenAddr(),
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	s.logStartupInfo()
	return s.server.ListenAndServe()
}

// logStartupInfo logs information about the server startup.
func (s *DevServer) logStartupInfo() {
	s.log.Infof("HTTP server listening on %s", s.listenAddr())
	s.log.Info("Available endpoints:")
	s.log.Info("  GET  /functions  - Discover available functions")
	s.log.Info("  POST /invoke     - Invoke a Forst function")
	s.log.Info("  GET  /types      - Generate TypeScript types for discovered functions")
	s.log.Info("  GET  /health     - Health check")
	s.log.Info("  GET  /version    - Compiler and HTTP contract version")
}

// Stop stops the HTTP server.
func (s *DevServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// discoverFunctions discovers all available functions.
func (s *DevServer) discoverFunctions() error {
	if s.discoverer == nil {
		return fmt.Errorf("discoverer not configured")
	}
	functions, err := s.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("failed to discover functions: %v", err)
	}

	s.mu.Lock()
	s.functions = functions
	s.mu.Unlock()

	return nil
}

// refreshFunctions updates invoke DevBackend discovery and mirrors into s.functions.
func (s *DevServer) refreshFunctions() error {
	if s.devBackend != nil {
		if err := s.devBackend.RefreshFunctions(context.Background()); err != nil {
			return fmt.Errorf("failed to discover functions: %v", err)
		}
		s.mu.Lock()
		s.functions = s.devBackend.Functions()
		s.mu.Unlock()
		return nil
	}
	return s.discoverFunctions()
}

// StartDevServer is the entry point for the dev server command.
func StartDevServer(port string, log *logrus.Logger, configPath string, rootDir string, logLevel *string, exportStructFieldsCLI bool, entryCLI string) error {
	config := loadAndValidateConfig(configPath, log, port, logLevel, rootDir, exportStructFieldsCLI)

	profile := devserver.ResolveProfile(&config.Config)
	log.Infof("forst dev: profile=%s", profile)

	if profile == devserver.ProfileRuntime {
		invokePort := devserver.EffectiveListenPort(&config.Config, port)
		_ = os.Setenv("FORST_INVOKE_PORT", invokePort)
		entry, err := devserver.ResolveEntry(rootDir, &config.Config, entryCLI)
		if err != nil {
			log.Error(err)
			return err
		}
		log.Infof("forst dev: runtime entry %s", entry)
		if devserver.RuntimeWatchEnabled(&config.Config) {
			return watchRuntimeDevFn(log, rootDir, entry, &config.Config)
		}
		if err := runRuntimeDevFn(log, rootDir, entry, &config.Config); err != nil {
			log.Error(err)
			return err
		}
		return nil
	}

	args := config.ToCompilerArgs()
	comp := compiler.New(args, log)

	listenPort := devserver.EffectiveListenPort(&config.Config, port)
	server := NewHTTPServer(listenPort, comp, log, config, rootDir)

	log.Debugf("Starting Forst dev server on %s", config.Server.EffectiveDevListenHost()+":"+listenPort)
	log.Debugf("Root directory: %s", rootDir)

	if err := devServerStartFn(server); err != nil {
		log.Errorf("HTTP server error: %v", err)
		return err
	}
	return nil
}

// devServerStartFn runs the HTTP server loop; tests may replace with a no-op.
var devServerStartFn = func(s *DevServer) error { return s.Start() }

// runRuntimeDevFn runs compile+go run for runtime profile; tests may stub.
var runRuntimeDevFn = func(log *logrus.Logger, boundaryRoot, entry string, cfg *ftconfig.Config) error {
	return devserver.RunRuntimeDev(log, boundaryRoot, entry, cfg, devserver.RuntimeRunDeps{})
}

// watchRuntimeDevFn runs compile+watch loop for runtime profile; tests may stub.
var watchRuntimeDevFn = func(log *logrus.Logger, boundaryRoot, entry string, cfg *ftconfig.Config) error {
	return devserver.WatchRuntimeDev(log, boundaryRoot, entry, cfg, devserver.RuntimeRunDeps{})
}

func loadAndValidateConfig(configPath string, log *logrus.Logger, port string, logLevel *string, rootDir string, exportStructFieldsCLI bool) *ForstConfig {
	resolvedConfigPath := configPath
	if resolvedConfigPath == "" && rootDir != "" {
		if found, _ := FindConfigFile(rootDir); found != "" {
			resolvedConfigPath = found
		}
	}

	config, err := LoadConfig(resolvedConfigPath)
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	if exportStructFieldsCLI {
		config.Compiler.ExportStructFields = true
	} else if !config.Compiler.ExportStructFields && rootDir != "" {
		config.Compiler.ExportStructFields = ftconfig.ExportStructFieldsFromDir(rootDir)
	}

	if resolvedConfigPath == "" {
		log.Infof("No config file provided, using default configuration")
	} else {
		log.Infof("Loaded config from: %s", resolvedConfigPath)
	}

	if port != "" && port != config.Server.Port {
		config.Server.Port = port
	}

	if logLevel != nil {
		config.Dev.LogLevel = *logLevel
	}

	type configSection struct {
		name    string
		entries []string
	}
	sections := []configSection{
		{"Server", []string{
			fmt.Sprintf("%-15s %s", "Port:", config.Server.Port),
			fmt.Sprintf("%-15s %s", "Host:", config.Server.EffectiveDevListenHost()),
			fmt.Sprintf("%-15s %v", "CORS enabled:", config.Server.CORS),
		}},
		{"Compiler", []string{
			fmt.Sprintf("%-15s %s", "Target:", config.Compiler.Target),
			fmt.Sprintf("%-15s %s", "Optimization:", config.Compiler.Optimization),
			fmt.Sprintf("%-15s %v", "Report phases:", config.Compiler.ReportPhases),
			fmt.Sprintf("%-15s %v", "Export struct fields (JSON):", config.Compiler.ExportStructFields),
		}},
		{"Files", []string{
			fmt.Sprintf("%-15s %s", "Include:", config.Files.Include),
			fmt.Sprintf("%-15s %s", "Exclude:", config.Files.Exclude),
		}},
		{"Output", []string{
			fmt.Sprintf("%-15s %s", "Dir:", config.Output.Dir),
			fmt.Sprintf("%-15s %s", "File name:", config.Output.FileName),
			fmt.Sprintf("%-15s %v", "Source maps:", config.Output.SourceMaps),
			fmt.Sprintf("%-15s %v", "Clean:", config.Output.Clean),
		}},
		{"Dev", []string{
			fmt.Sprintf("%-15s %s", "Profile:", config.Dev.Profile),
			fmt.Sprintf("%-15s %s", "Entry:", config.Dev.Entry),
			fmt.Sprintf("%-15s %v", "Hot reload:", config.Dev.HotReload),
			fmt.Sprintf("%-15s %v", "Watch:", config.Dev.Watch),
			fmt.Sprintf("%-15s %v", "Auto restart:", config.Dev.AutoRestart),
			fmt.Sprintf("%-15s %s", "Log level:", config.Dev.LogLevel),
			fmt.Sprintf("%-15s %v", "Verbose:", config.Dev.Verbose),
		}},
	}

	setLogLevel(log, config.Dev.LogLevel)

	for _, section := range sections {
		log.Debugf("%s:", section.name)
		for _, entry := range section.entries {
			log.Debugf("    %s", entry)
		}
	}

	if err := config.Validate(); err != nil {
		log.Errorf("Invalid configuration: %v", err)
		os.Exit(1)
	}

	return config
}
