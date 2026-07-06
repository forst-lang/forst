package main

import (
	"fmt"
	"slices"

	"forst/internal/compiler"
	"forst/internal/ftconfig"
)

// ForstConfig is the ftconfig.json schema with CLI adapter methods.
type ForstConfig struct {
	ftconfig.Config
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *ForstConfig {
	return &ForstConfig{Config: *ftconfig.Default()}
}

// FindConfigFile searches for ftconfig.json starting from the current directory
// and traversing upwards until the root directory, similar to TypeScript's behavior.
func FindConfigFile(startDir string) (string, error) {
	return ftconfig.FindConfigFile(startDir)
}

// LoadConfig loads configuration from a file or uses defaults.
func LoadConfig(configPath string) (*ForstConfig, error) {
	cfg, err := ftconfig.Load(configPath)
	if err != nil {
		return nil, err
	}
	return &ForstConfig{Config: *cfg}, nil
}

// ToCompilerArgs converts the config to compiler arguments.
func (c *ForstConfig) ToCompilerArgs() compiler.Args {
	return compiler.Args{
		LogLevel:           c.Dev.LogLevel,
		ReportPhases:       c.Compiler.ReportPhases,
		ReportMemoryUsage:  c.Compiler.ReportMemoryUsage,
		ExportStructFields: c.Compiler.ExportStructFields,
	}
}

// Validate validates the configuration for the dev server.
func (c *ForstConfig) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	validLogLevels := []string{"debug", "info", "warn", "error", "trace"}
	if !slices.Contains(validLogLevels, c.Dev.LogLevel) {
		return fmt.Errorf("invalid log level: %s", c.Dev.LogLevel)
	}
	return nil
}
