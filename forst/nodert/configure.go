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
	envNodeBootstrap = "FORST_NODE_BOOTSTRAP"
	envNodeBinary    = "FORST_NODE_BINARY"
	envNodeAppReadyModule = "FORST_NODE_APP_READY_MODULE"
	envNodeProtocolDefault = WireProtocolProtoV1
	// EnvBoundaryRoot is the ftconfig project root for Node interop.
	// `forst run -root …` sets this on the child Go process; it may also be set explicitly.
	EnvBoundaryRoot = "FORST_BOUNDARY_ROOT"
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
		if root := strings.TrimSpace(os.Getenv(EnvBoundaryRoot)); root != "" {
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

	nodeBinary, err := ResolveNodeBinary(workDir, cfg.Node.Binary)
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
		return fmt.Errorf("node runtime: resolve boundary root: %w", err)
	}
	manifest.BoundaryRoot = boundaryRoot
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("node runtime: manifest: %w", err)
	}

	if cfg.Node.HostMode && len(cfg.Node.Args) == 0 {
		return fmt.Errorf("node runtime: hostMode requires non-empty node.args in ftconfig.json")
	}

	hostProcessCfg, err := HostProcessConfigFromFTConfig(cfg, boundaryRoot, nil)
	if err != nil && cfg.Node.HostMode {
		return err
	}

	ConfigureSupervisor(SupervisorConfig{
		HostMode: cfg.Node.HostMode,
		HostSocketPath: hostProcessCfg.SocketPath,
		HostReadyPath:  hostProcessCfg.ReadyPath,
		HostReadyTimeout: hostProcessCfg.ReadyTimeout,
		HostAutoRegister: hostProcessCfg.HostAutoRegister,
		HostAppReadyModule: hostProcessCfg.HostAppReadyModule,
		ShimArgs: hostProcessCfg.ShimArgs,
		AttachOnly: os.Getenv(EnvNodeAttachOnly) == "1",
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
	if path := os.Getenv(envNodeBootstrap); path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("node runtime: bootstrap not found at %s: %w", path, err)
		}
		return path, nil
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
	for dir := startDir; ; dir = filepath.Dir(dir) {
		for _, base := range []string{dir, filepath.Dir(dir)} {
			candidate := filepath.Join(base, "packages", "node-runtime", "dist", "bootstrap.js")
			if st, statErr := os.Stat(candidate); statErr == nil && !st.IsDir() {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", fmt.Errorf("node runtime: bootstrap not found (set %s or node.bootstrap in ftconfig.json)", envNodeBootstrap)
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
	return "", fmt.Errorf("node runtime: host register.mjs not found (build packages/node-runtime or install @forst/node-runtime)")
}
