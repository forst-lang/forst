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
	Node     NodeConfig     `json:"node"`
}

// NodeRPCConfig represents Node stdio RPC limits.
type NodeRPCConfig struct {
	MaxMessageBytes    int `json:"maxMessageBytes"`
	CallTimeoutSeconds int `json:"callTimeoutSeconds"`
}

// NodeConfig represents TypeScript / Node runtime interop settings.
type NodeConfig struct {
	Enabled                 bool          `json:"enabled"`
	ImportPolicy            string        `json:"importPolicy"`
	RuntimeEnabled          bool          `json:"runtimeEnabled"`
	HostMode                bool          `json:"hostMode"`
	Binary                  string        `json:"binary"`
	Args                    []string      `json:"args"`
	HostSocket              string        `json:"hostSocket"`
	HostReadyTimeoutSeconds int           `json:"hostReadyTimeoutSeconds"`
	HostAutoRegister        *bool         `json:"hostAutoRegister,omitempty"`
	// HostAppReadyModule is an optional module to import before signaling nodert readiness.
	// Use a tiny side-effect-free module for third-party shims (e.g. remix-serve). Do not
	// point this at a full server bundle — import side effects can break host registration.
	HostAppReadyModule      string        `json:"hostAppReadyModule"`
	Bootstrap               string        `json:"bootstrap"`
	Loader                  string        `json:"loader"`
	GoRuntimeModule         string        `json:"goRuntimeModule"`
	GoRuntimeVersion        string        `json:"goRuntimeVersion"`
	RPC                     NodeRPCConfig `json:"rpc"`
}

// EffectiveHostAutoRegister reports whether nodert should inject host/register.mjs on spawn.
// Defaults to true when hostMode is enabled.
func (n NodeConfig) EffectiveHostAutoRegister() bool {
	if n.HostAutoRegister != nil {
		return *n.HostAutoRegister
	}
	return n.HostMode
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
	// Embedded enables an in-process invoke HTTP server in compiled Go binaries.
	Embedded       bool   `json:"embedded"`
	Port           string `json:"port"`
	Host           string `json:"host"`
	CORS           bool   `json:"cors"`
	ReadTimeout    int    `json:"readTimeout"`
	WriteTimeout   int    `json:"writeTimeout"`
	MaxRequestSize int64  `json:"maxRequestSize"`
}

// EffectiveInvokeHost returns the bind host for embedded invoke.
// Embedded node-to-forst RPC always listens on loopback only.
func (s ServerConfig) EffectiveInvokeHost() string {
	if s.Embedded {
		return "127.0.0.1"
	}
	if s.Host == "" {
		return "localhost"
	}
	return s.Host
}

// EffectiveDevListenHost returns the bind host for forst dev.
// Empty or "localhost" defaults to loopback; explicit values (e.g. 0.0.0.0) are preserved.
func (s ServerConfig) EffectiveDevListenHost() string {
	if s.Host == "" || s.Host == "localhost" {
		return "127.0.0.1"
	}
	return s.Host
}

// EffectiveInvokePort returns the listen port; embedded defaults to 8081 to avoid clashing with forst dev (8080).
func (s ServerConfig) EffectiveInvokePort() string {
	if s.Port != "" {
		return s.Port
	}
	if s.Embedded {
		return "8081"
	}
	return "8080"
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
		Node: NodeConfig{
			Enabled:        false,
			ImportPolicy:   "explicit",
			RuntimeEnabled: false,
			Binary:         "node",
			Bootstrap:      "node_modules/@forst/node-runtime/dist/bootstrap.js",
			Loader:         "tsx",
			HostSocket:     ".forst/node.sock",
			HostReadyTimeoutSeconds: 120,
			RPC: NodeRPCConfig{
				MaxMessageBytes:    16 << 20,
				CallTimeoutSeconds: 120,
			},
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
	normalizeNodeConfig(config)
	if err := validateNodeConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

// BoundaryRootFromDir returns the directory containing ftconfig.json found by walking upward from startDir.
func BoundaryRootFromDir(startDir string) (string, error) {
	path, err := FindConfigFile(startDir)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("%s not found from %s", configFileName, startDir)
	}
	return filepath.Clean(filepath.Dir(path)), nil
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

// ImportPolicyFromDir returns node.importPolicy from ftconfig.json found by walking
// upward from startDir, or "explicit" when no config is found or the field is empty.
func ImportPolicyFromDir(startDir string) string {
	cfg, err := LoadFromDir(startDir)
	if err != nil || cfg == nil {
		return "explicit"
	}
	if cfg.Node.ImportPolicy == "" {
		return "explicit"
	}
	return cfg.Node.ImportPolicy
}

func normalizeServerMaxRequestSize(config *Config) {
	if config.Server.MaxRequestSize <= 0 {
		config.Server.MaxRequestSize = httpbody.DefaultMaxBytes
	}
}

func normalizeNodeConfig(config *Config) {
	if config.Node.Binary == "" {
		config.Node.Binary = "node"
	}
	if config.Node.Bootstrap == "" {
		config.Node.Bootstrap = "node_modules/@forst/node-runtime/dist/bootstrap.js"
	}
	if config.Node.Loader == "" {
		config.Node.Loader = "tsx"
	}
	if config.Node.RPC.MaxMessageBytes <= 0 {
		config.Node.RPC.MaxMessageBytes = 16 << 20
	}
	if config.Node.RPC.CallTimeoutSeconds <= 0 {
		config.Node.RPC.CallTimeoutSeconds = 120
	}
	if config.Node.HostSocket == "" {
		config.Node.HostSocket = ".forst/node.sock"
	}
	if config.Node.HostReadyTimeoutSeconds <= 0 {
		config.Node.HostReadyTimeoutSeconds = 120
	}
}

func validateNodeConfig(config *Config) error {
	if config.Node.HostMode && len(config.Node.Args) == 0 {
		return fmt.Errorf("node.hostMode requires non-empty node.args")
	}
	return nil
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
	defer func() { _ = root.Close() }()

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
