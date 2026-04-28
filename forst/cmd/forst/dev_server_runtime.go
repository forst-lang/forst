package main

import (
	"fmt"
	"os"

	"forst/internal/compiler"
	"forst/internal/devserver"

	logrus "github.com/sirupsen/logrus"
)

// StartDevServer is the entry point for the dev server command.
func StartDevServer(port string, log *logrus.Logger, configPath string, rootDir string, logLevel *string) error {
	config := loadAndValidateConfig(configPath, log, port, logLevel)

	args := config.ToCompilerArgs()
	comp := compiler.New(args, log)

	server := devserver.NewHTTPServer(
		config.Server.Port,
		comp,
		log,
		config,
		devserver.BuildInfo{Version: Version, Commit: Commit, Date: Date},
		devserver.HTTPOpts{
			ReadTimeoutSec:  config.Server.ReadTimeout,
			WriteTimeoutSec: config.Server.WriteTimeout,
			CORS:            config.Server.CORS,
		},
		rootDir,
	)

	log.Debugf("Starting Forst dev server on port %s", config.Server.Port)
	log.Debugf("Root directory: %s", rootDir)

	if err := devServerStartFn(server); err != nil {
		log.Errorf("HTTP server error: %v", err)
		return err
	}
	return nil
}

// devServerStartFn runs the HTTP server loop; tests may replace with a no-op.
var devServerStartFn = func(s *devserver.DevServer) error { return s.Start() }

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
