package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"forst/internal/compiler"

	logrus "github.com/sirupsen/logrus"
)

// Start starts the HTTP server.
func (s *DevServer) Start() error {
	if err := s.discoverFunctions(); err != nil {
		s.log.Warnf("Failed to discover functions on startup: %v", err)
	}

	s.typesGenerator = NewTypeScriptGenerator(s.log)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/version", s.handleVersion)
	mux.HandleFunc("/functions", s.handleFunctions)
	mux.HandleFunc("/invoke", s.handleInvoke)
	mux.HandleFunc("/types", s.handleTypes)

	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  time.Duration(s.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Server.WriteTimeout) * time.Second,
	}

	s.logStartupInfo()
	return s.server.ListenAndServe()
}

// logStartupInfo logs information about the server startup.
func (s *DevServer) logStartupInfo() {
	s.log.Infof("HTTP server listening on port %s", s.port)
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
	functions, err := s.discoverer.DiscoverFunctions()
	if err != nil {
		return fmt.Errorf("failed to discover functions: %v", err)
	}

	s.mu.Lock()
	s.functions = functions
	s.mu.Unlock()

	return nil
}

// StartDevServer is the entry point for the dev server command.
func StartDevServer(port string, log *logrus.Logger, configPath string, rootDir string, logLevel *string) error {
	config := loadAndValidateConfig(configPath, log, port, logLevel)

	args := config.ToCompilerArgs()
	comp := compiler.New(args, log)

	server := NewHTTPServer(config.Server.Port, comp, log, config, rootDir)

	log.Debugf("Starting Forst dev server on port %s", config.Server.Port)
	log.Debugf("Root directory: %s", rootDir)

	if err := server.Start(); err != nil {
		log.Errorf("HTTP server error: %v", err)
		return err
	}
	return nil
}

func loadAndValidateConfig(configPath string, log *logrus.Logger, port string, logLevel *string) *ForstConfig {
	config, err := LoadConfig(configPath)
	if err != nil {
		log.Errorf("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	if configPath == "" {
		log.Infof("No config file provided, using default configuration")
	} else {
		log.Infof("Loaded config from: %s", configPath)
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
