package nodert

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
<<<<<<< Updated upstream:forst/nodert/configure.go
	envNodeBootstrap       = "FORST_NODE_BOOTSTRAP"
	envNodeBinary          = "FORST_NODE_BINARY"
	envNodeAppReadyModule  = "FORST_NODE_APP_READY_MODULE"
	envNodeProtocolDefault = WireProtocolProtoV1
=======
	envBridgeBootstrap       = "FORST_BRIDGE_BOOTSTRAP"
	envBridgeBinary          = "FORST_BRIDGE_BINARY"
	envBridgeAppReadyModule  = "FORST_BRIDGE_APP_READY_MODULE"
	envBridgeProtocolDefault = WireProtocolProtoV1
>>>>>>> Stashed changes:forst/bridgert/configure.go
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
		return bridgeRuntimeErr("empty manifest JSON")
	}
	var manifest Manifest
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return bridgeRuntimeErr("parse manifest: %w", err)
	}
	if err := manifest.ValidateEmbedded(); err != nil {
		return bridgeRuntimeErr("invalid embedded manifest: %w", err)
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
			return bridgeRuntimeErr("getwd: %w", err)
		}
		workDir, err = ftconfig.BoundaryRootFromDir(cwd)
		if err != nil {
			return bridgeRuntimeErr("discover boundary root: %w", err)
		}
	}

	cfg, err := ftconfig.LoadFromDir(workDir)
	if err != nil {
		return bridgeRuntimeErr("load ftconfig: %w", err)
	}

<<<<<<< Updated upstream:forst/nodert/configure.go
	nodeBinary, err := ResolveNodeBinary(workDir, cfg.Node.Binary)
=======
	nodeBinary, err := ResolveBridgeBinary(workDir, cfg.Bridge.Binary)
>>>>>>> Stashed changes:forst/bridgert/configure.go
	if err != nil {
		return err
	}

	var bootstrap string
	if !cfg.Node.HostMode {
		bootstrap, err = ResolveBootstrapPath(workDir, cfg.Node.Bootstrap)
		if err != nil {
			return err
		}
	}

	boundaryRoot, err := filepath.Abs(workDir)
	if err != nil {
		return bridgeRuntimeErr("resolve boundary root: %w", err)
	}
	manifest.BoundaryRoot = boundaryRoot
	if err := manifest.Validate(); err != nil {
		return bridgeRuntimeErr("manifest: %w", err)
	}

<<<<<<< Updated upstream:forst/nodert/configure.go
	effectiveHostMode := cfg.Node.HostMode && !SkipNodeHostEnabled()
	if effectiveHostMode && len(cfg.Node.Args) == 0 {
		return fmt.Errorf("node runtime: hostMode requires non-empty node.args in ftconfig.json")
=======
	effectiveHostMode := cfg.Bridge.HostMode && !SkipNodeHostEnabled()
	if effectiveHostMode && len(cfg.Bridge.Args) == 0 {
		return bridgeHostErr("hostMode requires non-empty bridge.args in ftconfig.json")
>>>>>>> Stashed changes:forst/bridgert/configure.go
	}

	hostProcessCfg, err := HostProcessConfigFromFTConfig(cfg, boundaryRoot, nil)
	if err != nil && effectiveHostMode {
		return err
	}

<<<<<<< Updated upstream:forst/nodert/configure.go
=======
	bridge, err := ftconfig.EffectiveBridge(cfg)
	if err != nil {
		return bridgeRuntimeErr("%w", err)
	}
	moduleIDs := manifestModuleIDs(manifest)

	modulesDir := ""
	if bridge.ModuleFormat == ftconfig.LegacyModuleCompiled && manifestUsesCompiledModules(manifest) {
		modulesDir, err = ftconfig.ResolveModulesDir(boundaryRoot, cfg)
		if err != nil {
			return bridgeRuntimeErr("resolve compiled modules directory: %w", err)
		}
		if err := validateCompiledModulesDir(modulesDir); err != nil {
			return err
		}
	}

>>>>>>> Stashed changes:forst/bridgert/configure.go
	ConfigureSupervisor(SupervisorConfig{
		HostMode: effectiveHostMode,
		HostSocketPath: hostProcessCfg.SocketPath,
		HostReadyPath:  hostProcessCfg.ReadyPath,
		HostReadyTimeout: hostProcessCfg.ReadyTimeout,
		HostAutoRegister: hostProcessCfg.HostAutoRegister,
		HostAppReadyModule: hostProcessCfg.HostAppReadyModule,
		ShimArgs: hostProcessCfg.ShimArgs,
<<<<<<< Updated upstream:forst/nodert/configure.go
		AttachOnly: os.Getenv(EnvNodeAttachOnly) == "1",
=======
		AttachOnly: os.Getenv(EnvBridgeAttachOnly) == "1",
		ModulesDir: modulesDir,
>>>>>>> Stashed changes:forst/bridgert/configure.go
		ProcessOptions: ProcessOptions{
			NodePath:      nodeBinary,
			BootstrapPath: bootstrap,
			WorkDir:       workDir,
			Loader:        cfg.Node.Loader,
			BoundaryRoot:  boundaryRoot,
			FilesExclude:  append([]string(nil), cfg.Files.Exclude...),
		},
		Manifest: manifest,
		RPC: RPCConfig{
			MaxMessageBytes: cfg.Node.RPC.MaxMessageBytes,
			CallTimeout:     time.Duration(cfg.Node.RPC.CallTimeoutSeconds) * time.Second,
		},
	})
	return nil
}

// ResolveBootstrapPath returns the Node bootstrap script path.
// Priority: FORST_NODE_BOOTSTRAP env → ftconfig.node.bootstrap (relative to boundaryRoot) → monorepo walk-up.
func ResolveBootstrapPath(boundaryRoot, configuredBootstrap string) (string, error) {
	if path := strings.TrimSpace(os.Getenv(envBridgeBootstrap)); path != "" {
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
<<<<<<< Updated upstream:forst/nodert/configure.go
		return "", fmt.Errorf("node runtime: bootstrap not found at %s (set %s to an absolute path or build packages/node-runtime)", path, envNodeBootstrap)
=======
		return "", bridgeRuntimeErr("bootstrap not found at %s (set %s to an absolute path or build packages/runtime)", path, envBridgeBootstrap)
>>>>>>> Stashed changes:forst/bridgert/configure.go
	}

	if configuredBootstrap != "" {
		candidate := configuredBootstrap
		if !filepath.IsAbs(candidate) {
			if boundaryRoot == "" {
				var err error
				boundaryRoot, err = os.Getwd()
				if err != nil {
					return "", bridgeRuntimeErr("%w", err)
				}
			}
			candidate = filepath.Join(boundaryRoot, configuredBootstrap)
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			return "", bridgeRuntimeErr("resolve bootstrap path: %w", err)
		}
		if st, statErr := os.Stat(abs); statErr != nil {
			return "", bridgeRuntimeErr("bootstrap not found at %s: %w", abs, statErr)
		} else if st.IsDir() {
			return "", bridgeRuntimeErr("bootstrap path is a directory: %s", abs)
		}
		return abs, nil
	}

	startDir := boundaryRoot
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", bridgeRuntimeErr("%w", err)
		}
	}
	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	if abs, ok := findMonorepoBootstrap(startDir); ok {
		return abs, nil
	}
	return "", bridgeRuntimeErr("bootstrap not found (set %s or bridge.bootstrap in ftconfig.json)", envBridgeBootstrap)
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
			candidate := filepath.Join(base, "packages", "node-runtime", "dist", "bootstrap.js")
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
		return "", bridgeRuntimeErr("resolve hostAppReadyModule: %w", err)
	}
	if st, statErr := os.Stat(abs); statErr != nil {
		return "", bridgeRuntimeErr("hostAppReadyModule not found at %s: %w", abs, statErr)
	} else if st.IsDir() {
		return "", bridgeRuntimeErr("hostAppReadyModule is a directory: %s", abs)
	}
	return abs, nil
}

// ResolveHostRegisterPath returns the blocking host register.mjs preload script.
func ResolveHostRegisterPath(boundaryRoot string) (string, error) {
	if boundaryRoot == "" {
		var err error
		boundaryRoot, err = os.Getwd()
		if err != nil {
			return "", bridgeRuntimeErr("%w", err)
		}
	}
	boundaryRoot, err := filepath.Abs(boundaryRoot)
	if err != nil {
		return "", err
	}

	candidate := filepath.Join(boundaryRoot, "node_modules", "@forst", "node-runtime", "dist", "host", "register.mjs")
	if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
		return candidate, nil
	}

	for dir := boundaryRoot; ; dir = filepath.Dir(dir) {
		for _, base := range []string{dir, filepath.Dir(dir)} {
			candidate = filepath.Join(base, "packages", "node-runtime", "dist", "host", "register.mjs")
			if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
<<<<<<< Updated upstream:forst/nodert/configure.go
	return "", fmt.Errorf("node runtime: host register.mjs not found (build packages/node-runtime or install @forst/node-runtime)")
=======
	return "", bridgeHostErr("host register.mjs not found (build packages/runtime or install @forst/runtime)")
}

func validateCompiledModulesDir(modulesDir string) error {
	if strings.TrimSpace(modulesDir) == "" {
		return bridgeRuntimeErr(
			"compiled modules directory is not configured (set %s or bridge.legacyModules.dir in ftconfig.json)",
			ftconfig.EnvBridgeModulesDir,
		)
	}
	st, err := os.Stat(modulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return bridgeRuntimeErr(
			"compiled modules directory %q does not exist (copy or mount .forst/js and set %s if not at the default location)",
				modulesDir,
				ftconfig.EnvBridgeModulesDir,
			)
		}
		return bridgeRuntimeErr("compiled modules directory %q: %w", modulesDir, err)
	}
	if !st.IsDir() {
		return bridgeRuntimeErr("compiled modules path %q is not a directory", modulesDir)
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
>>>>>>> Stashed changes:forst/bridgert/configure.go
}
