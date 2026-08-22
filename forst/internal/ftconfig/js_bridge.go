package ftconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// JSHost is the JavaScript bridge interpreter (node, bun, deno).
type JSHost string

const (
	JSHostNode JSHost = "node"
	JSHostBun  JSHost = "bun"
	JSHostDeno JSHost = "deno"
)

// LegacyModuleFormat controls whether legacy modules load from compile-time JS bundles or TypeScript source.
type LegacyModuleFormat string

const (
	LegacyModuleCompiled   LegacyModuleFormat = "compiled"
	LegacyModuleTypeScript LegacyModuleFormat = "typescript"
)

// JavascriptConfig is the optional overlay for bridge host and legacy module loading.
type JavascriptConfig struct {
	Host          JSHost              `json:"host"`
	LegacyModules LegacyModulesConfig `json:"legacyModules"`
}

// LegacyModulesConfig configures compile-time precompile and runtime module format.
type LegacyModulesConfig struct {
	Format     LegacyModuleFormat `json:"format"`
	Artifact   LegacyModuleFormat `json:"artifact,omitempty"` // deprecated: use format
	Precompile PrecompileConfig   `json:"precompile"`
}

// PrecompileConfig controls esbuild output for compiled legacy modules.
type PrecompileConfig struct {
	OutDir string `json:"outDir"`
	Tool   string `json:"tool"`
	Format string `json:"format"`
}

// JSBridge is the effective bridge host + module format after merging ftconfig.
type JSBridge struct {
	Host         JSHost
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

// EffectiveJSBridge merges javascript.* overlay, deprecated node.loader, and node.binary inference.
func EffectiveJSBridge(cfg *Config) (JSBridge, error) {
	if cfg == nil {
		return JSBridge{}, fmt.Errorf("ftconfig: config is nil")
	}

	host := cfg.Javascript.Host
	if host == "" {
		host = InferHostFromBinary(cfg.Node.Binary)
	}
	if host == "" {
		host = JSHostNode
	}
	if err := validateJSHost(host); err != nil {
		return JSBridge{}, err
	}

	moduleFormat := effectiveLegacyModuleFormat(cfg)
	loader := strings.TrimSpace(cfg.Node.Loader)

	switch loader {
	case "":
		// no legacy loader mapping
	case "tsx":
		if host != JSHostNode {
			return JSBridge{}, fmt.Errorf("node.loader tsx is only valid when javascript.host is node (got %q)", host)
		}
		if moduleFormat == "" {
			moduleFormat = LegacyModuleTypeScript
		}
	case "none":
		if moduleFormat == "" {
			moduleFormat = LegacyModuleCompiled
		}
	default:
		return JSBridge{}, fmt.Errorf("unsupported node.loader %q (deprecated; use javascript.legacyModules.format)", loader)
	}

	if moduleFormat == "" {
		moduleFormat = LegacyModuleCompiled
	}
	if err := validateLegacyModuleFormat(moduleFormat); err != nil {
		return JSBridge{}, err
	}

	outDir := strings.TrimSpace(cfg.Javascript.LegacyModules.Precompile.OutDir)
	if outDir == "" {
		outDir = ".forst/js"
	}
	outDir = filepath.ToSlash(filepath.Clean(outDir))
	if strings.HasPrefix(outDir, "/") || strings.Contains(outDir, "..") {
		return JSBridge{}, fmt.Errorf("javascript.legacyModules.precompile.outDir must be a relative project path, got %q", outDir)
	}

	if host == JSHostDeno && !denoHostEnabled() {
		return JSBridge{}, fmt.Errorf("javascript.host deno is not enabled yet (set FORST_DENO_HOST_ENABLED=1 when supported)")
	}

	return JSBridge{
		Host:         host,
		ModuleFormat: moduleFormat,
		OutDir:       outDir,
	}, nil
}

func effectiveLegacyModuleFormat(cfg *Config) LegacyModuleFormat {
	lm := cfg.Javascript.LegacyModules
	if lm.Format != "" {
		return normalizeLegacyModuleFormat(lm.Format)
	}
	if lm.Artifact != "" {
		return normalizeLegacyModuleFormat(lm.Artifact)
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
func NeedTsx(bridge JSBridge, moduleIDs []string) bool {
	if bridge.Host != JSHostNode {
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

// InferHostFromBinary guesses JSHost from node.binary basename.
func InferHostFromBinary(binary string) JSHost {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(binary)))
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "bun":
		return JSHostBun
	case "deno":
		return JSHostDeno
	default:
		return JSHostNode
	}
}

func validateJSHost(host JSHost) error {
	switch host {
	case JSHostNode, JSHostBun, JSHostDeno:
		return nil
	default:
		return fmt.Errorf("unsupported javascript.host %q (want node, bun, or deno)", host)
	}
}

func validateLegacyModuleFormat(mode LegacyModuleFormat) error {
	switch normalizeLegacyModuleFormat(mode) {
	case LegacyModuleCompiled, LegacyModuleTypeScript:
		return nil
	default:
		return fmt.Errorf("unsupported javascript.legacyModules.format %q (want compiled or typescript)", mode)
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
