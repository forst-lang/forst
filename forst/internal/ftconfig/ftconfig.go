package ftconfig

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/configiface"
	"forst/internal/httpbody"
	"forst/internal/safefs"

	"github.com/bmatcuk/doublestar/v4"
)

const configFileName = "ftconfig.json"

// Config represents the configuration for the Forst dev server (ftconfig.json).
type Config struct {
	Compiler CompilerConfig `json:"compiler"`
	Server   ServerConfig   `json:"server"`
	Files    FilesConfig    `json:"files"`
	Output   OutputConfig   `json:"output"`
	Dev      DevConfig      `json:"dev"`
}

// CompilerConfig represents compiler-specific settings.
type CompilerConfig struct {
	Target                   string `json:"target"`
	Optimization             string `json:"optimization"`
	ReportPhases             bool   `json:"reportPhases"`
	ReportMemoryUsage        bool   `json:"reportMemoryUsage"`
	Strict                   bool   `json:"strict"`
	ExportStructFields       bool   `json:"exportStructFields"`
	GenerateStreamingClients bool   `json:"generateStreamingClients"`
}

// ServerConfig represents HTTP server settings.
type ServerConfig struct {
	Port           string `json:"port"`
	Host           string `json:"host"`
	CORS           bool   `json:"cors"`
	ReadTimeout    int    `json:"readTimeout"`
	WriteTimeout   int    `json:"writeTimeout"`
	MaxRequestSize int64  `json:"maxRequestSize"`
}

// FilesConfig represents file discovery settings.
type FilesConfig struct {
	Include  []string `json:"include"`
	Exclude  []string `json:"exclude"`
	MaxDepth int      `json:"maxDepth"`
}

// OutputConfig represents output settings.
type OutputConfig struct {
	Dir        string `json:"dir"`
	FileName   string `json:"fileName"`
	SourceMaps bool   `json:"sourceMaps"`
	Clean      bool   `json:"clean"`
}

// DevConfig represents development-specific settings.
type DevConfig struct {
	HotReload   bool   `json:"hotReload"`
	Watch       bool   `json:"watch"`
	AutoRestart bool   `json:"autoRestart"`
	LogLevel    string `json:"logLevel"`
	Verbose     bool   `json:"verbose"`
}

var _ configiface.ForstConfigIface = (*Config)(nil)

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Compiler: CompilerConfig{
			Target:             "go",
			Optimization:       "debug",
			ReportPhases:       false,
			ReportMemoryUsage:  false,
			Strict:             false,
			ExportStructFields: false,
		},
		Server: ServerConfig{
			Port:           "8080",
			Host:           "localhost",
			CORS:           true,
			ReadTimeout:    30,
			WriteTimeout:   30,
			MaxRequestSize: 10 * 1024 * 1024,
		},
		Files: FilesConfig{
			Include:  []string{"**/*.ft"},
			Exclude:  []string{"**/node_modules/**", "**/.git/**"},
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

// FindConfigFile searches upward from startDir for ftconfig.json.
func FindConfigFile(startDir string) (string, error) {
	currentDir := startDir
	for {
		configPath := filepath.Join(currentDir, configFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}
	return "", nil
}

// Load loads configuration from a file path, or discovers ftconfig.json when path is empty.
func Load(configPath string) (*Config, error) {
	config := Default()

	if configPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			if foundPath, err := FindConfigFile(cwd); err == nil && foundPath != "" {
				configPath = foundPath
			}
		}
	}

	if configPath != "" {
		if err := loadFromFile(configPath, config); err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %v", configPath, err)
		}
	}

	normalizeServerMaxRequestSize(config)
	return config, nil
}

// LoadFromDir walks upward from startDir for ftconfig.json and loads it, or returns defaults.
func LoadFromDir(startDir string) (*Config, error) {
	path, err := FindConfigFile(startDir)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return Default(), nil
	}
	return Load(path)
}

// ExportStructFieldsFromDir returns compiler.exportStructFields from ftconfig.json
// found by walking upward from startDir, or false when no config is found.
func ExportStructFieldsFromDir(startDir string) bool {
	cfg, err := LoadFromDir(startDir)
	if err != nil || cfg == nil {
		return false
	}
	return cfg.Compiler.ExportStructFields
}

func normalizeServerMaxRequestSize(config *Config) {
	if config.Server.MaxRequestSize <= 0 {
		config.Server.MaxRequestSize = httpbody.DefaultMaxBytes
	}
}

func loadFromFile(configPath string, config *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}
	return nil
}

// FindForstFiles recursively finds all .ft files in the configured directories under rootDir.
func (c *Config) FindForstFiles(rootDir string) ([]string, error) {
	root, err := safefs.OpenRoot(rootDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	var files []string
	err = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		absPath := root.AbsPath(path)
		if !strings.HasSuffix(strings.ToLower(absPath), ".ft") {
			return nil
		}
		if !c.matchesIncludePatterns(absPath) {
			return nil
		}
		if c.matchesExcludePatterns(absPath) {
			return nil
		}
		files = append(files, absPath)
		return nil
	})
	return files, err
}

func (c *Config) matchesIncludePatterns(path string) bool {
	if len(c.Files.Include) == 0 {
		return true
	}
	for _, pattern := range c.Files.Include {
		if c.matchesPattern(path, pattern) {
			return true
		}
	}
	return false
}

func (c *Config) matchesExcludePatterns(path string) bool {
	for _, pattern := range c.Files.Exclude {
		if c.matchesPattern(path, pattern) {
			return true
		}
	}
	return false
}

func (c *Config) matchesPattern(path, pattern string) bool {
	matched, err := doublestar.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}
