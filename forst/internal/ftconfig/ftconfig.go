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

// EnvRoot is the ftconfig project root for runtime discovery (.forst/, invoke.ready, bridgert).
// `forst run -root …` sets this on child processes; it may also be set explicitly.
const EnvRoot = "FORST_ROOT"

// RootFromEnv returns the project root from FORST_ROOT when set.
func RootFromEnv() string {
	return strings.TrimSpace(os.Getenv(EnvRoot))
}

// Config represents the configuration for the Forst dev server (ftconfig.json).
type Config struct {
	Compiler CompilerConfig `json:"compiler"`
	Server   ServerConfig   `json:"server"`
	Files    FilesConfig    `json:"files"`
	Output   OutputConfig   `json:"output"`
	Dev      DevConfig      `json:"dev"`
	Bridge   BridgeConfig   `json:"bridge"`
	Generate   GenerateConfig   `json:"generate"`
}

// GenerateConfig controls TypeScript client package generation (forst generate).
type GenerateConfig struct {
	PackageName    string           `json:"packageName"`
	OutDir         string           `json:"outDir"`
	Link           string           `json:"link"`
	Emit           string           `json:"emit"`
	TestingSubpath string           `json:"testingSubpath"`
	Effect         bool             `json:"effect"`
	SSRModule      string           `json:"ssrModule"`
	Go             GenerateGoConfig `json:"go"`
	// SkipClient skips TypeScript client generation (Go-only or custom pipelines).
	SkipClient bool `json:"skipClient"`
	// OmitStubs emits commented stubs for provider-gated omissions in package modules (SPEC §12).
	OmitStubs bool `json:"omitStubs"`
	// Plugins lists local semantic plugin executables run after typecheck (forst generate only).
	Plugins []GeneratePluginConfig `json:"plugins"`
}

// GeneratePluginConfig configures one semantic plugin runner entry.
type GeneratePluginConfig struct {
	Name string          `json:"name"`
	Cmd  string          `json:"cmd"`
	Out  string          `json:"out"`
	Opt  json.RawMessage `json:"opt,omitempty"`
}

// GenerateGoConfig controls optional Go source emission from forst generate.
// Go emission is active when both entry and out are set.
type GenerateGoConfig struct {
	Entry string `json:"entry"`
	Out   string `json:"out"`
	Root  string `json:"root"`
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

// EffectiveInvokePort returns the listen port; embedded defaults to 6321 to avoid clashing with forst dev (6320).
func (s ServerConfig) EffectiveInvokePort() string {
	if s.Port != "" {
		return s.Port
	}
	if s.Embedded {
		return DefaultEmbeddedInvokePort
	}
	return DefaultDevExecutorPort
}

// FilesConfig represents file discovery settings.
// Include/exclude globs are resolved from the ftconfig boundary root. Package layout still
// requires one directory per Forst package name for the Go target (see forstpkg.ValidateOneDirectoryPerPackage).
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
	Profile     string `json:"profile"` // auto | executor | runtime
	Entry       string `json:"entry"`   // optional runtime dev entry .ft (relative to boundary)
	HotReload   bool   `json:"hotReload"`
	Watch       bool   `json:"watch"`
	AutoRestart bool   `json:"autoRestart"`
	LogLevel    string `json:"logLevel"`
	Verbose     bool   `json:"verbose"`
	// WatchGenerate runs forst generate after debounced reloads so editor types stay in step.
	// Defaults to true when unset.
	WatchGenerate *bool `json:"watchGenerate,omitempty"`
}

// EffectiveWatchGenerate reports whether forst dev should regenerate the TypeScript client on reload.
func (d DevConfig) EffectiveWatchGenerate() bool {
	if d.WatchGenerate != nil {
		return *d.WatchGenerate
	}
	return true
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
			Port:           DefaultDevExecutorPort,
			Host:           "localhost",
			CORS:           true,
			ReadTimeout:    30,
			WriteTimeout:   30,
			MaxRequestSize: 10 * 1024 * 1024,
		},
		Files: FilesConfig{
			Include:  []string{"**/*.ft"},
			Exclude:  []string{"**/node_modules/**", "**/.git/**", "**/build/**", "**/.forst/**"},
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
		Bridge: BridgeConfig{
			Enabled:        false,
			ImportPolicy:   "explicit",
			RuntimeEnabled: false,
			Host:           BridgeHostNode,
			Binary:         "node",
			Bootstrap:      "node_modules/@forst/runtime/dist/bootstrap.js",
			HostSocket:     ".forst/bridge.sock",
			HostReadyTimeoutSeconds: 120,
			RPC: BridgeRPCConfig{
				MaxMessageBytes:    16 << 20,
				CallTimeoutSeconds: 120,
			},
		},
		Generate: defaultGenerateConfig(),
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
	normalizeBridgeConfig(config)
	if err := validateBridgeConfig(config); err != nil {
		return nil, err
	}
	if _, err := EffectiveBridge(config); err != nil {
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

// ImportPolicyFromDir returns bridge.importPolicy from ftconfig.json found by walking
// upward from startDir, or "explicit" when no config is found or the field is empty.
func ImportPolicyFromDir(startDir string) string {
	cfg, err := LoadFromDir(startDir)
	if err != nil || cfg == nil {
		return "explicit"
	}
	if cfg.Bridge.ImportPolicy == "" {
		return "explicit"
	}
	return cfg.Bridge.ImportPolicy
}

func normalizeServerMaxRequestSize(config *Config) {
	if config.Server.MaxRequestSize <= 0 {
		config.Server.MaxRequestSize = httpbody.DefaultMaxBytes
	}
}

func normalizeBridgeConfig(config *Config) {
	if config.Bridge.Binary == "" {
		config.Bridge.Binary = "node"
	}
	if config.Bridge.Bootstrap == "" {
		config.Bridge.Bootstrap = "node_modules/@forst/runtime/dist/bootstrap.js"
	}
	if config.Bridge.RPC.MaxMessageBytes <= 0 {
		config.Bridge.RPC.MaxMessageBytes = 16 << 20
	}
	if config.Bridge.RPC.CallTimeoutSeconds <= 0 {
		config.Bridge.RPC.CallTimeoutSeconds = 120
	}
	if config.Bridge.HostSocket == "" {
		config.Bridge.HostSocket = ".forst/bridge.sock"
	}
	if config.Bridge.HostReadyTimeoutSeconds <= 0 {
		config.Bridge.HostReadyTimeoutSeconds = 120
	}
}

func validateBridgeConfig(config *Config) error {
	if config.Bridge.HostMode && len(config.Bridge.Args) == 0 {
		return fmt.Errorf("bridge.hostMode requires non-empty bridge.args")
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
			if path != "." {
				absPath := root.AbsPath(path)
				if c.matchesExcludePatterns(absPath) {
					return fs.SkipDir
				}
			}
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

// MatchesIncludePatterns reports whether path matches configured include globs.
func (c *Config) MatchesIncludePatterns(path string) bool {
	return c.matchesIncludePatterns(path)
}

// MatchesExcludePatterns reports whether path matches configured exclude globs.
func (c *Config) MatchesExcludePatterns(path string) bool {
	return c.matchesExcludePatterns(path)
}

func (c *Config) matchesPattern(path, pattern string) bool {
	matched, err := doublestar.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}
