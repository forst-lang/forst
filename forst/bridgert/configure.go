package bridgert

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"forst/internal/ftconfig"
)

const (
	envNodeBootstrap       = "FORST_BRIDGE_BOOTSTRAP"
	envNodeBinary          = "FORST_BRIDGE_BINARY"
	envNodeAppReadyModule  = "FORST_BRIDGE_APP_READY_MODULE"
	envNodeProtocolDefault = WireProtocolProtoV1
	// EnvRoot is the ftconfig project root for Node interop.
	EnvRoot = ftconfig.EnvRoot
)

var (
	configureOnce sync.Once
	configureErr  error
)

// MustConfigureFromManifest parses embedded manifest JSON and configures the supervisor.
// Generated Go programs call this from init() when needsNodeRuntime is true.
func MustConfigureFromManifest(manifestJSON string) {
	configureOnce.Do(func() {
		configureErr = configureFromManifest(manifestJSON)
	})
	if configureErr != nil {
		panic(configureErr)
	}
}

func configureFromManifest(manifestJSON string) error {
	if manifestJSON == "" {
		return fmt.Errorf("node runtime: empty manifest JSON")
	}
	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return fmt.Errorf("node runtime: parse manifest: %w", err)
	}
	if err := manifest.ValidateEmbedded(); err != nil {
		return fmt.Errorf("node runtime: invalid embedded manifest: %w", err)
	}

	workDir := strings.TrimSpace(manifest.BoundaryRoot)
	if workDir == "" {
		if root := ftconfig.RootFromEnv(); root != "" {
			workDir = root
		}
	}
	if workDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("node runtime: getwd: %w", err)
		}
		workDir, err = ftconfig.BoundaryRootFromDir(cwd)
		if err != nil {
			return fmt.Errorf("node runtime: discover boundary root: %w", err)
		}
	}

	cfg, err := ftconfig.LoadFromDir(workDir)
	if err != nil {
		return fmt.Errorf("node runtime: load ftconfig: %w", err)
	}

	nodeBinary, err := ResolveNodeBinary(workDir, cfg.Bridge.Binary)
	if err != nil {
		return err
	}

	var bootstrap string
	if !cfg.Bridge.HostMode {
		bootstrap, err = ResolveBootstrapPath(workDir, cfg.Bridge.Bootstrap)
		if err != nil {
			return err
		}
	}

	boundaryRoot, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("node runtime: resolve boundary root: %w", err)
	}
	manifest.BoundaryRoot = boundaryRoot
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("node runtime: manifest: %w", err)
	}

	effectiveHostMode := cfg.Bridge.HostMode && !SkipNodeHostEnabled()
	if effectiveHostMode && len(cfg.Bridge.Args) == 0 {
		return fmt.Errorf("node runtime: hostMode requires non-empty bridge.args in ftconfig.json")
	}

	hostProcessCfg, err := HostProcessConfigFromFTConfig(cfg, boundaryRoot, nil)
	if err != nil && effectiveHostMode {
		return err
	}

	bridge, err := ftconfig.EffectiveBridge(cfg)
	if err != nil {
		return fmt.Errorf("node runtime: %w", err)
	}
	moduleIDs := manifestModuleIDs(manifest)

	modulesDir := ""
	if bridge.ModuleFormat == ftconfig.LegacyModuleCompiled && manifestUsesCompiledModules(manifest) {
		modulesDir, err = ftconfig.ResolveModulesDir(boundaryRoot, cfg)
		if err != nil {
			return fmt.Errorf("node runtime: resolve compiled modules directory: %w", err)
		}
		if err := validateCompiledModulesDir(modulesDir); err != nil {
			return err
		}
	}

	ConfigureSupervisor(SupervisorConfig{
		HostMode: effectiveHostMode,
		HostSocketPath: hostProcessCfg.SocketPath,
		HostReadyPath:  hostProcessCfg.ReadyPath,
		HostReadyTimeout: hostProcessCfg.ReadyTimeout,
		HostAutoRegister: hostProcessCfg.HostAutoRegister,
		HostAppReadyModule: hostProcessCfg.HostAppReadyModule,
		ShimArgs: hostProcessCfg.ShimArgs,
		AttachOnly: os.Getenv(EnvNodeAttachOnly) == "1",
		ModulesDir: modulesDir,
		ProcessOptions: ProcessOptions{
			NodePath:      nodeBinary,
			BootstrapPath: bootstrap,
			WorkDir:       workDir,
			Bridge:        bridge,
			ModuleIDs:     moduleIDs,
			BoundaryRoot:  boundaryRoot,
			ModulesDir:    modulesDir,
			FilesExclude:  append([]string(nil), cfg.Files.Exclude...),
		},
		Manifest: manifest,
		RPC: RPCConfig{
			MaxMessageBytes: cfg.Bridge.RPC.MaxMessageBytes,
			CallTimeout:     time.Duration(cfg.Bridge.RPC.CallTimeoutSeconds) * time.Second,
		},
	})
	return nil
}

// ResolveBootstrapPath returns the Node bootstrap script path.
// Priority: FORST_BRIDGE_BOOTSTRAP env → ftconfig.bridge.bootstrap (relative to boundaryRoot) → monorepo walk-up.
func ResolveBootstrapPath(boundaryRoot, configuredBootstrap string) (string, error) {
	if path := strings.TrimSpace(os.Getenv(envNodeBootstrap)); path != "" {
		if abs, err := firstExistingBootstrap(resolveBootstrapCandidates(boundaryRoot, path)...); err == nil {
			return abs, nil
		}
		if abs, ok := findMonorepoBootstrap(boundaryRoot); ok {
			return abs, nil
		}
		if cwd, err := os.Getwd(); err == nil {
			if abs, ok := findMonorepoBootstrap(cwd); ok {
				return abs, nil
			}
		}
		return "", fmt.Errorf("node runtime: bootstrap not found at %s (set %s to an absolute path or build packages/runtime)", path, envNodeBootstrap)
	}

	if configuredBootstrap != "" {
		candidate := configuredBootstrap
		if !filepath.IsAbs(candidate) {
			if boundaryRoot == "" {
				var err error
				boundaryRoot, err = os.Getwd()
				if err != nil {
					return "", fmt.Errorf("node runtime: %w", err)
				}
			}
			candidate = filepath.Join(boundaryRoot, configuredBootstrap)
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			return "", fmt.Errorf("node runtime: resolve bootstrap path: %w", err)
		}
		if st, statErr := os.Stat(abs); statErr != nil {
			return "", fmt.Errorf("node runtime: bootstrap not found at %s: %w", abs, statErr)
		} else if st.IsDir() {
			return "", fmt.Errorf("node runtime: bootstrap path is a directory: %s", abs)
		}
		return abs, nil
	}

	startDir := boundaryRoot
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("node runtime: %w", err)
		}
	}
	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	if abs, ok := findMonorepoBootstrap(startDir); ok {
		return abs, nil
	}
	return "", fmt.Errorf("node runtime: bootstrap not found (set %s or node.bootstrap in ftconfig.json)", envNodeBootstrap)
}

func resolveBootstrapCandidates(boundaryRoot, path string) []string {
	seen := make(map[string]struct{})
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
	}
	add(path)
	if !filepath.IsAbs(path) {
		if boundaryRoot != "" {
			add(filepath.Join(boundaryRoot, path))
		}
		if cwd, err := os.Getwd(); err == nil {
			add(filepath.Join(cwd, path))
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	return out
}

func firstExistingBootstrap(candidates ...string) (string, error) {
	for _, candidate := range candidates {
		st, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if st.IsDir() {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("bootstrap not found")
}

func findMonorepoBootstrap(startDir string) (string, bool) {
	if startDir == "" {
		return "", false
	}
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return "", false
	}
	for dir := absStart; ; dir = filepath.Dir(dir) {
		for _, base := range []string{dir, filepath.Dir(dir)} {
			candidate := filepath.Join(base, "packages", "runtime", "dist", "bootstrap.js")
			if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
				abs, err := filepath.Abs(candidate)
				if err != nil {
					return candidate, true
				}
				return abs, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", false
}

func resolveHostAppReadyModule(boundaryRoot, configured string) (string, error) {
	candidate := configured
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(boundaryRoot, configured)
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("node runtime: resolve hostAppReadyModule: %w", err)
	}
	if st, statErr := os.Stat(abs); statErr != nil {
		return "", fmt.Errorf("node runtime: hostAppReadyModule not found at %s: %w", abs, statErr)
	} else if st.IsDir() {
		return "", fmt.Errorf("node runtime: hostAppReadyModule is a directory: %s", abs)
	}
	return abs, nil
}

// ResolveHostRegisterPath returns the blocking host register.mjs preload script.
func ResolveHostRegisterPath(boundaryRoot string) (string, error) {
	if boundaryRoot == "" {
		var err error
		boundaryRoot, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("node runtime: %w", err)
		}
	}
	boundaryRoot, err := filepath.Abs(boundaryRoot)
	if err != nil {
		return "", err
	}

	candidate := filepath.Join(boundaryRoot, "node_modules", "@forst", "runtime", "dist", "host", "register.mjs")
	if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
		return candidate, nil
	}

	for dir := boundaryRoot; ; dir = filepath.Dir(dir) {
		for _, base := range []string{dir, filepath.Dir(dir)} {
			candidate = filepath.Join(base, "packages", "runtime", "dist", "host", "register.mjs")
			if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("node runtime: host register.mjs not found (build packages/runtime or install @forst/runtime)")
}

func validateCompiledModulesDir(modulesDir string) error {
	if strings.TrimSpace(modulesDir) == "" {
		return fmt.Errorf(
			"node runtime: compiled modules directory is not configured (set %s or bridge.legacyModules.dir in ftconfig.json)",
			ftconfig.EnvBridgeModulesDir,
		)
	}
	st, err := os.Stat(modulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(
				"node runtime: compiled modules directory %q does not exist (copy or mount .forst/js and set %s if not at the default location)",
				modulesDir,
				ftconfig.EnvBridgeModulesDir,
			)
		}
		return fmt.Errorf("node runtime: compiled modules directory %q: %w", modulesDir, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("node runtime: compiled modules path %q is not a directory", modulesDir)
	}
	return nil
}

func manifestUsesCompiledModules(m Manifest) bool {
	for _, exp := range m.Exports {
		ext := strings.ToLower(filepath.Ext(strings.TrimSpace(exp.ModuleID)))
		if ext == ".js" {
			return true
		}
	}
	return false
}

func manifestModuleIDs(m Manifest) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, exp := range m.Exports {
		id := strings.TrimSpace(exp.ModuleID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
