package nodert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	envNodeHost      = "FORST_NODE_HOST"
	envNodeSocket    = "FORST_NODE_SOCKET"
	envNodeHostReady = "FORST_NODE_HOST_READY"
)

// mergeNodeOptions appends import flags idempotently to existing NODE_OPTIONS.
func mergeNodeOptions(existing string, additions ...string) string {
	result := strings.TrimSpace(existing)
	for _, addition := range additions {
		addition = strings.TrimSpace(addition)
		if addition == "" {
			continue
		}
		if result == "" {
			result = addition
			continue
		}
		if strings.Contains(result, addition) {
			continue
		}
		result += " " + addition
	}
	return result
}

func lookupEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}

// ResolveNodeBinary returns the Node or shim executable path.
// Priority: FORST_NODE_BINARY env → configured → "node".
func ResolveNodeBinary(boundaryRoot, configured string) (string, error) {
	candidate := configured
	if path := os.Getenv(envNodeBinary); path != "" {
		candidate = path
	}
	if candidate == "" {
		candidate = "node"
	}
	return resolveExecutablePath(boundaryRoot, candidate)
}

func resolveExecutablePath(boundaryRoot, candidate string) (string, error) {
	if candidate == "" {
		return "", fmt.Errorf("node runtime: executable path is empty")
	}

	resolved := candidate
	if !filepath.IsAbs(resolved) && !isBareExecutableName(resolved) {
		if boundaryRoot == "" {
			var err error
			boundaryRoot, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("node runtime: getwd: %w", err)
			}
		}
		resolved = filepath.Join(boundaryRoot, resolved)
	}

	if isBareExecutableName(resolved) {
		if path, err := lookPathExecutable(resolved); err == nil {
			return path, nil
		}
		if runtime.GOOS == "windows" {
			for _, suffix := range []string{".cmd", ".exe"} {
				if path, err := lookPathExecutable(resolved + suffix); err == nil {
					return path, nil
				}
			}
		}
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("node runtime: resolve executable path: %w", err)
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		eval = abs
	}
	st, err := os.Stat(eval)
	if err != nil {
		return "", fmt.Errorf("node runtime: executable not found at %s: %w", eval, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("node runtime: executable path is a directory: %s", eval)
	}
	return eval, nil
}

func isBareExecutableName(name string) bool {
	return !strings.Contains(name, string(os.PathSeparator)) && !strings.Contains(name, "/")
}

func lookPathExecutable(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("node runtime: executable not found at %s", name)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path, nil
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs, nil
	}
	return eval, nil
}

// ResolveHostSocketPath returns the absolute Unix socket path under boundaryRoot.
func ResolveHostSocketPath(boundaryRoot, configured string) (string, string, error) {
	if path := strings.TrimSpace(os.Getenv(envNodeSocket)); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", fmt.Errorf("node runtime: resolve host socket: %w", err)
		}
		return abs, abs + ".ready", nil
	}
	if configured == "" {
		configured = ".forst/node.sock"
	}
	if filepath.IsAbs(configured) {
		return "", "", fmt.Errorf("node runtime: hostSocket must be relative to boundary root")
	}
	clean := filepath.Clean(configured)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || strings.Contains(clean, "..") {
		return "", "", fmt.Errorf("node runtime: hostSocket escapes boundary: %q", configured)
	}
	if boundaryRoot == "" {
		var err error
		boundaryRoot, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("node runtime: getwd: %w", err)
		}
	}
	socketPath, err := filepath.Abs(filepath.Join(boundaryRoot, clean))
	if err != nil {
		return "", "", fmt.Errorf("node runtime: resolve host socket: %w", err)
	}
	return socketPath, socketPath + ".ready", nil
}

// PrepareHostSocket removes stale socket and ready marker files.
func PrepareHostSocket(socketPath, readyPath string) error {
	if socketPath != "" {
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("node runtime: remove stale host socket: %w", err)
		}
	}
	if readyPath != "" {
		if err := os.Remove(readyPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("node runtime: remove stale host ready file: %w", err)
		}
	}
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("node runtime: create host socket dir: %w", err)
	}
	return nil
}

// BootstrapSpawnInput configures bootstrap-mode child spawn.
type BootstrapSpawnInput struct {
	BoundaryRoot  string
	Executable    string
	BootstrapPath string
	WorkDir       string
	Loader        string
	FilesExclude  []string
	Env           []string
	ExtraArgs     []string
}

// BootstrapSpawnCommand is the resolved bootstrap spawn invocation.
type BootstrapSpawnCommand struct {
	Executable string
	Args       []string
	Env        []string
}

// BuildBootstrapSpawnCommand builds argv/env for dedicated bootstrap child.
func BuildBootstrapSpawnCommand(in BootstrapSpawnInput) (BootstrapSpawnCommand, error) {
	executable, err := ResolveNodeBinary(in.BoundaryRoot, in.Executable)
	if err != nil {
		return BootstrapSpawnCommand{}, err
	}
	if in.BootstrapPath == "" {
		return BootstrapSpawnCommand{}, fmt.Errorf("bootstrap path is required")
	}
	bootstrapPath, err := filepath.Abs(in.BootstrapPath)
	if err != nil {
		return BootstrapSpawnCommand{}, fmt.Errorf("resolve bootstrap path: %w", err)
	}
	if _, err := os.Stat(bootstrapPath); err != nil {
		return BootstrapSpawnCommand{}, fmt.Errorf("bootstrap not found at %s: %w", bootstrapPath, err)
	}

	loader := in.Loader
	if loader == "" {
		loader = "tsx"
	}
	var tsxImport string
	switch loader {
	case "tsx":
		tsxLoader, err := ResolveTsxLoaderPath(bootstrapPath, in.WorkDir)
		if err != nil {
			return BootstrapSpawnCommand{}, err
		}
		tsxImport = "--import " + tsxLoader
	default:
		return BootstrapSpawnCommand{}, fmt.Errorf("unsupported node loader %q", loader)
	}

	env := buildSpawnEnv(spawnEnvInput{
		BoundaryRoot: in.BoundaryRoot,
		FilesExclude: in.FilesExclude,
		Env:          in.Env,
		NodeOptions:  []string{tsxImport},
	})

	args := append([]string{bootstrapPath}, in.ExtraArgs...)
	return BootstrapSpawnCommand{
		Executable: executable,
		Args:       args,
		Env:        env,
	}, nil
}

// HostSpawnInput configures host-mode app shim spawn.
type HostSpawnInput struct {
	BoundaryRoot       string
	Executable         string
	ShimArgs           []string
	WorkDir            string
	Loader             string
	SocketPath         string
	ReadyPath          string
	FilesExclude       []string
	Env                []string
	HostAutoRegister   bool
	HostAppReadyModule string
}

// HostSpawnCommand is the resolved host-mode spawn invocation.
type HostSpawnCommand struct {
	Executable string
	Args       []string
	Env        []string
	SocketPath string
	ReadyPath  string
}

type spawnEnvInput struct {
	BoundaryRoot string
	FilesExclude []string
	Env          []string
	NodeOptions  []string
	HostMode     bool
	SocketPath   string
	ReadyPath    string
}

func buildSpawnEnv(in spawnEnvInput) []string {
	opts := ProcessOptions{
		BoundaryRoot: in.BoundaryRoot,
		FilesExclude: in.FilesExclude,
		Env:          in.Env,
	}
	env := buildNodeChildEnv(opts)
	existing := lookupEnvValue(env, "NODE_OPTIONS")
	merged := mergeNodeOptions(existing, in.NodeOptions...)
	env = setEnvVar(env, "NODE_OPTIONS", merged)
	if in.HostMode {
		env = setEnvVar(env, envNodeHost, "1")
		if in.SocketPath != "" {
			env = setEnvVar(env, envNodeSocket, in.SocketPath)
		}
		if in.ReadyPath != "" {
			env = setEnvVar(env, envNodeHostReady, in.ReadyPath)
		}
	}
	return env
}

// BuildHostSpawnCommand builds argv/env for app shim spawn in host mode.
func BuildHostSpawnCommand(in HostSpawnInput) (HostSpawnCommand, error) {
	if len(in.ShimArgs) == 0 {
		return HostSpawnCommand{}, fmt.Errorf("node runtime: hostMode requires non-empty node.args")
	}
	executable, err := ResolveNodeBinary(in.BoundaryRoot, in.Executable)
	if err != nil {
		return HostSpawnCommand{}, err
	}
	socketPath := in.SocketPath
	readyPath := in.ReadyPath
	if socketPath == "" || readyPath == "" {
		var err error
		socketPath, readyPath, err = ResolveHostSocketPath(in.BoundaryRoot, "")
		if err != nil {
			return HostSpawnCommand{}, err
		}
	}

	loader := in.Loader
	if loader == "" {
		loader = "tsx"
	}
	var tsxImport string
	switch loader {
	case "tsx":
		tsxLoader, err := ResolveTsxLoaderPath(in.BoundaryRoot, in.WorkDir)
		if err != nil {
			return HostSpawnCommand{}, err
		}
		tsxImport = "--import " + tsxLoader
	default:
		return HostSpawnCommand{}, fmt.Errorf("unsupported node loader %q", loader)
	}

	nodeOptions := []string{tsxImport}
	childEnv := append([]string(nil), in.Env...)
	if in.HostAutoRegister {
		registerPath, err := ResolveHostRegisterPath(in.BoundaryRoot)
		if err != nil {
			return HostSpawnCommand{}, err
		}
		nodeOptions = append(nodeOptions, "--import "+registerPath)
	}
	if in.HostAppReadyModule != "" {
		childEnv = setEnvVar(childEnv, envNodeAppReadyModule, in.HostAppReadyModule)
	}

	env := buildSpawnEnv(spawnEnvInput{
		BoundaryRoot: in.BoundaryRoot,
		FilesExclude: in.FilesExclude,
		Env:          childEnv,
		NodeOptions:  nodeOptions,
		HostMode:     true,
		SocketPath:   socketPath,
		ReadyPath:    readyPath,
	})

	return HostSpawnCommand{
		Executable: executable,
		Args:       append([]string(nil), in.ShimArgs...),
		Env:        env,
		SocketPath: socketPath,
		ReadyPath:  readyPath,
	}, nil
}
