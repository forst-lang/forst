package ftconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BridgeHost is the JavaScript bridge interpreter (node, bun, deno).
type BridgeHost string

const (
	BridgeHostNode BridgeHost = "node"
	BridgeHostBun  BridgeHost = "bun"
	BridgeHostDeno BridgeHost = "deno"
)

// LegacyModuleFormat controls whether legacy modules load from compile-time JS bundles or TypeScript source.
type LegacyModuleFormat string

const (
	LegacyModuleCompiled   LegacyModuleFormat = "compiled"
	LegacyModuleTypeScript LegacyModuleFormat = "typescript"
)

// LegacyModulesConfig configures compile-time precompile and runtime module format.
type LegacyModulesConfig struct {
	Format     LegacyModuleFormat `json:"format"`
	Precompile PrecompileConfig   `json:"precompile"`
}

// PrecompileConfig controls esbuild output for compiled legacy modules.
type PrecompileConfig struct {
	OutDir string `json:"outDir"`
	Tool   string `json:"tool"`
	Format string `json:"format"`
}

// BridgeRPCConfig represents bridge stdio/socket RPC limits.
type BridgeRPCConfig struct {
	MaxMessageBytes    int `json:"maxMessageBytes"`
	CallTimeoutSeconds int `json:"callTimeoutSeconds"`
}

// BridgeConfig represents JavaScript bridge interop settings.
type BridgeConfig struct {
	Enabled                 bool                `json:"enabled"`
	ImportPolicy            string              `json:"importPolicy"`
	RuntimeEnabled          bool                `json:"runtimeEnabled"`
	Host                    BridgeHost          `json:"host"`
	LegacyModules           LegacyModulesConfig `json:"legacyModules"`
	HostMode                bool                `json:"hostMode"`
	Binary                  string              `json:"binary"`
	Args                    []string            `json:"args"`
	HostSocket              string              `json:"hostSocket"`
	HostReadyTimeoutSeconds int                 `json:"hostReadyTimeoutSeconds"`
	HostAutoRegister        *bool               `json:"hostAutoRegister,omitempty"`
	HostAppReadyModule      string              `json:"hostAppReadyModule"`
	Bootstrap               string              `json:"bootstrap"`
	GoRuntimeModule         string              `json:"goRuntimeModule"`
	GoRuntimeVersion        string              `json:"goRuntimeVersion"`
	RPC                     BridgeRPCConfig     `json:"rpc"`
}

// EffectiveHostAutoRegister reports whether bridgert should inject host/register.mjs on spawn.
// Defaults to true when hostMode is enabled.
func (b BridgeConfig) EffectiveHostAutoRegister() bool {
	if b.HostAutoRegister != nil {
		return *b.HostAutoRegister
	}
	return b.HostMode
}

// Bridge is the effective bridge host + module format after merging ftconfig.
type Bridge struct {
	Host         BridgeHost
	ModuleFormat LegacyModuleFormat
	OutDir       string // POSIX relative, default ".forst/js"
}

var denoHostEnabledOverride *bool

// SetDenoHostEnabledForTest toggles deno host support in unit tests.
func SetDenoHostEnabledForTest(enabled bool) {
	denoHostEnabledOverride = &enabled
}

func denoHostEnabled() bool {
	if denoHostEnabledOverride != nil {
		return *denoHostEnabledOverride
	}
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FORST_DENO_HOST_ENABLED")))
	return v == "1" || v == "true"
}

// EffectiveBridge resolves bridge.host, legacy module format, and precompile out dir.
func EffectiveBridge(cfg *Config) (Bridge, error) {
	if cfg == nil {
		return Bridge{}, fmt.Errorf("ftconfig: config is nil")
	}

	host := cfg.Bridge.Host
	if host == "" {
		host = InferHostFromBinary(cfg.Bridge.Binary)
	}
	if host == "" {
		host = BridgeHostNode
	}
	if err := validateBridgeHost(host); err != nil {
		return Bridge{}, err
	}

	moduleFormat := effectiveLegacyModuleFormat(cfg)
	if moduleFormat == "" {
		moduleFormat = LegacyModuleCompiled
	}
	if err := validateLegacyModuleFormat(moduleFormat); err != nil {
		return Bridge{}, err
	}

	outDir := strings.TrimSpace(cfg.Bridge.LegacyModules.Precompile.OutDir)
	if outDir == "" {
		outDir = ".forst/js"
	}
	outDir = filepath.ToSlash(filepath.Clean(outDir))
	if strings.HasPrefix(outDir, "/") || strings.Contains(outDir, "..") {
		return Bridge{}, fmt.Errorf("bridge.legacyModules.precompile.outDir must be a relative project path, got %q", outDir)
	}

	if host == BridgeHostDeno && !denoHostEnabled() {
		return Bridge{}, fmt.Errorf("bridge.host deno is not enabled yet (set FORST_DENO_HOST_ENABLED=1 when supported)")
	}

	return Bridge{
		Host:         host,
		ModuleFormat: moduleFormat,
		OutDir:       outDir,
	}, nil
}

func effectiveLegacyModuleFormat(cfg *Config) LegacyModuleFormat {
	lm := cfg.Bridge.LegacyModules
	if lm.Format != "" {
		return normalizeLegacyModuleFormat(lm.Format)
	}
	return ""
}

func normalizeLegacyModuleFormat(v LegacyModuleFormat) LegacyModuleFormat {
	switch LegacyModuleFormat(strings.TrimSpace(string(v))) {
	case LegacyModuleTypeScript, "source":
		return LegacyModuleTypeScript
	case LegacyModuleCompiled, "precompiled":
		return LegacyModuleCompiled
	default:
		return v
	}
}

// NeedTsx reports whether Node should inject tsx for the given module IDs and bridge settings.
func NeedTsx(bridge Bridge, moduleIDs []string) bool {
	if bridge.Host != BridgeHostNode {
		return false
	}
	for _, id := range moduleIDs {
		ext := strings.ToLower(filepath.Ext(id))
		if ext == ".ts" || ext == ".tsx" {
			return true
		}
	}
	if bridge.ModuleFormat == LegacyModuleCompiled {
		return false
	}
	return len(moduleIDs) == 0
}

// InferHostFromBinary guesses BridgeHost from bridge.binary basename.
func InferHostFromBinary(binary string) BridgeHost {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(binary)))
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "bun":
		return BridgeHostBun
	case "deno":
		return BridgeHostDeno
	default:
		return BridgeHostNode
	}
}

func validateBridgeHost(host BridgeHost) error {
	switch host {
	case BridgeHostNode, BridgeHostBun, BridgeHostDeno:
		return nil
	default:
		return fmt.Errorf("unsupported bridge.host %q (want node, bun, or deno)", host)
	}
}

func validateLegacyModuleFormat(mode LegacyModuleFormat) error {
	switch normalizeLegacyModuleFormat(mode) {
	case LegacyModuleCompiled, LegacyModuleTypeScript:
		return nil
	default:
		return fmt.Errorf("unsupported bridge.legacyModules.format %q (want compiled or typescript)", mode)
	}
}

// RuntimeModuleID maps a source moduleId to the runtime path for the given module format.
func RuntimeModuleID(sourceID, outDir string, format LegacyModuleFormat) string {
	if normalizeLegacyModuleFormat(format) != LegacyModuleCompiled {
		return sourceID
	}
	ext := filepath.Ext(sourceID)
	stem := strings.TrimSuffix(sourceID, ext)
	return filepath.ToSlash(filepath.Join(outDir, stem+".js"))
}
