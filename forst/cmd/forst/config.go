package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"forst/cmd/forst/compiler"
	"forst/internal/configiface"
)

// ForstConfig represents the configuration for the Forst dev server
// Similar to tsconfig.json format but in Go structs
type ForstConfig struct {
	// Compiler settings
	Compiler CompilerConfig `json:"compiler"`

	// Server settings
	Server ServerConfig `json:"server"`

	// File discovery settings
	Files FilesConfig `json:"files"`

	// Output settings
	Output OutputConfig `json:"output"`

	// Development settings
	Dev DevConfig `json:"dev"`
}

// CompilerConfig represents compiler-specific settings
type CompilerConfig struct {
	// Target language (go, wasm, etc.)
	Target string `json:"target"`

	// Optimization level (debug, release)
	Optimization string `json:"optimization"`

	// Enable debug output
	Debug bool `json:"debug"`

	// Enable trace output
	Trace bool `json:"trace"`

	// Report compilation phases
	ReportPhases bool `json:"reportPhases"`

	// Report memory usage
	ReportMemoryUsage bool `json:"reportMemoryUsage"`

	// Strict mode
	Strict bool `json:"strict"`
}

// ServerConfig represents HTTP server settings
type ServerConfig struct {
	// Port to listen on
	Port string `json:"port"`

	// Host to bind to
	Host string `json:"host"`

	// Enable CORS
	CORS bool `json:"cors"`

	// Read timeout in seconds
	ReadTimeout int `json:"readTimeout"`

	// Write timeout in seconds
	WriteTimeout int `json:"writeTimeout"`

	// Max request size in bytes
	MaxRequestSize int64 `json:"maxRequestSize"`
}

// FilesConfig represents file discovery settings
type FilesConfig struct {
	// Include patterns
	Include []string `json:"include"`

	// Exclude patterns
	Exclude []string `json:"exclude"`

	// Maximum directory depth
	MaxDepth int `json:"maxDepth"`
}

// OutputConfig represents output settings
type OutputConfig struct {
	// Output directory
	Dir string `json:"dir"`

	// Output file name pattern
	FileName string `json:"fileName"`

	// Generate source maps
	SourceMaps bool `json:"sourceMaps"`

	// Clean output directory before compilation
	Clean bool `json:"clean"`
}

// DevConfig represents development-specific settings
type DevConfig struct {
	// Enable hot reloading
	HotReload bool `json:"hotReload"`

	// Watch for file changes
	Watch bool `json:"watch"`

	// Auto-restart on file changes
	AutoRestart bool `json:"autoRestart"`

	// Log level (debug, info, warn, error)
	LogLevel string `json:"logLevel"`

	// Enable verbose logging
	Verbose bool `json:"verbose"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *ForstConfig {
	return &ForstConfig{
		Compiler: CompilerConfig{
			Target:            "go",
			Optimization:      "debug",
			Debug:             false,
			Trace:             false,
			ReportPhases:      false,
			ReportMemoryUsage: false,
			Strict:            false,
		},
		Server: ServerConfig{
			Port:           "8080",
			Host:           "localhost",
			CORS:           true,
			ReadTimeout:    30,
			WriteTimeout:   30,
			MaxRequestSize: 10 * 1024 * 1024, // 10MB
		},
		Files: FilesConfig{
			Include:  []string{"**/*.ft"},
			Exclude:  []string{"**/node_modules/**", "**/.git/**", "**/stream.ft", "**/test_*.ft", "**/*.skip.ft"},
			MaxDepth: 10,
		},
		Output: OutputConfig{
			Dir:        "dist",
			FileName:   "{{name}}.go",
			SourceMaps: false,
			Clean:      true,
		},
		Dev: DevConfig{
			HotReload:   true,
			Watch:       false,
			AutoRestart: true,
			LogLevel:    "info",
			Verbose:     false,
		},
	}
}

// FindConfigFile searches for ftconfig.json starting from the current directory
// and traversing upwards until the root directory, similar to TypeScript's behavior
func FindConfigFile(startDir string) (string, error) {
	currentDir := startDir

	for {
		configPath := filepath.Join(currentDir, "ftconfig.json")

		// Check if config file exists
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Move to parent directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// We've reached the root directory
			break
		}
		currentDir = parentDir
	}

	// No config file found
	return "", nil
}

// LoadConfig loads configuration from a file or uses defaults
func LoadConfig(configPath string) (*ForstConfig, error) {
	config := DefaultConfig()

	// If no config path provided, try to find one
	if configPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			if foundPath, err := FindConfigFile(cwd); err == nil && foundPath != "" {
				configPath = foundPath
			}
		}
	}

	// Load config file if found
	if configPath != "" {
		if err := loadConfigFromFile(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %v", configPath, err)
		}
	}

	return config, nil
}

// loadConfigFromFile loads configuration from a JSON file
func loadConfigFromFile(configPath string, config *ForstConfig) error {
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	return nil
}

// FindForstFiles recursively finds all .ft files in the configured directories
func (c *ForstConfig) FindForstFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file has .ft extension
		if !strings.HasSuffix(strings.ToLower(path), ".ft") {
			return nil
		}

		// Check include patterns
		if !c.matchesIncludePatterns(path) {
			return nil
		}

		// Check exclude patterns
		if c.matchesExcludePatterns(path) {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// matchesIncludePatterns checks if a file path matches any include patterns
func (c *ForstConfig) matchesIncludePatterns(path string) bool {
	if len(c.Files.Include) == 0 {
		return true // No include patterns means include everything
	}

	for _, pattern := range c.Files.Include {
		if c.matchesPattern(path, pattern) {
			return true
		}
	}

	return false
}

// matchesExcludePatterns checks if a file path matches any exclude patterns
func (c *ForstConfig) matchesExcludePatterns(path string) bool {
	for _, pattern := range c.Files.Exclude {
		if c.matchesPattern(path, pattern) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a file path matches a glob pattern
func (c *ForstConfig) matchesPattern(path, pattern string) bool {
	// Simple glob pattern matching
	// This is a basic implementation - in production you might want to use a proper glob library

	// Convert pattern to regex-like matching
	pattern = strings.ReplaceAll(pattern, "**", ".*")
	pattern = strings.ReplaceAll(pattern, "*", "[^/]*")
	pattern = strings.ReplaceAll(pattern, "?", ".")

	// Add start/end anchors
	pattern = "^" + pattern + "$"

	// Simple string matching for now
	// In a real implementation, you'd use regex or a proper glob library
	return strings.Contains(path, strings.ReplaceAll(pattern, ".*", ""))
}

// ToCompilerArgs converts the config to compiler arguments
func (c *ForstConfig) ToCompilerArgs() compiler.Args {
	return compiler.Args{
		Debug:             c.Compiler.Debug,
		Trace:             c.Compiler.Trace,
		ReportPhases:      c.Compiler.ReportPhases,
		ReportMemoryUsage: c.Compiler.ReportMemoryUsage,
	}
}

// Validate validates the configuration
func (c *ForstConfig) Validate() error {
	// Validate server port
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	logLevelValid := false
	for _, level := range validLogLevels {
		if c.Dev.LogLevel == level {
			logLevelValid = true
			break
		}
	}
	if !logLevelValid {
		return fmt.Errorf("invalid log level: %s", c.Dev.LogLevel)
	}

	return nil
}

var _ configiface.ForstConfigIface = (*ForstConfig)(nil)
