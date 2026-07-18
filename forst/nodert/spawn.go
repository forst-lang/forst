package nodert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Host-mode environment variable names.
//
// Go sets FORST_NODE_HOST, FORST_NODE_HOST_LEADER, FORST_NODE_SOCKET, and
// FORST_NODE_HOST_READY on the direct app-shim child when ftconfig node.hostMode
// is true (BuildHostSpawnCommand → buildSpawnEnv). @forst/node-runtime reads them
// in host/register.mjs and host.ts. Child processes spawned by the shim (Vite
// workers, cluster forks, etc.) may inherit the values but must not act as host
// leaders — see FORST_NODE_HOST_LEADER and register preload via argv, not
// NODE_OPTIONS. FORST_NODE_ATTACH_ONLY is separate: forst dev sets it on the
// parent process for watch-reload go run children (see below).
//
//   FORST_NODE_HOST
//     Gate for host RPC in Node. When "1", startForstNodeHost() may bind the RPC
//     socket; when unset, host code no-ops. Set only in host mode; bootstrap mode
//     uses stdio RPC and does not set this variable.
//
//   FORST_NODE_HOST_LEADER
//     Marks the process Go spawned as the sole host leader. startForstNodeHost()
//     requires "1" and register.mjs in process.execArgv; workers that inherit
//     FORST_NODE_HOST without leader/preload skip binding (host_skip_non_leader).
//     Do not set manually except for tests.
//
//   FORST_NODE_SOCKET
//     Absolute path to the Unix domain socket (loopback TCP URL on Windows) where
//     the in-process host listens for Go RPC. Defaults from node.hostSocket under
//     the boundary root (.forst/node.sock). Go dials this after readiness; may
//     also be read at spawn planning time via ResolveHostSocketPath when set in
//     the parent environment.
//
//   FORST_NODE_HOST_READY
//     Absolute path to a JSON readiness marker (typically socketPath + ".ready").
//     The host writes {"pid", "socket", "phase"} after listen and/or app init;
//     Go polls until phase is "app" before connecting. Used to avoid dialing
//     before third-party shims (Remix, Vite) finish bootstrapping when
//     hostAppReadyModule or signalForstAppReady() defer readiness.
//
//   FORST_NODE_ATTACH_ONLY
//     Attach-only gate for Go-side host supervision. When "1", nodert dials an
//     existing host via FORST_NODE_SOCKET / FORST_NODE_HOST_READY but never
//     spawns a new shim. forst dev sets this on the parent after EnsureRunning
//     so each watch-reload go run child inherits it; one-shot forst run leaves
//     it unset so the binary may spawn the host on first GetClient(). If no
//     live host is reachable, GetClient fails with an attach-only error instead
//     of starting a second Vite process.
const (
	envNodeHost       = "FORST_NODE_HOST"
	envNodeHostLeader = "FORST_NODE_HOST_LEADER"
	envNodeSocket     = "FORST_NODE_SOCKET"
	envNodeHostReady  = "FORST_NODE_HOST_READY"
	envNodeAttachOnly = "FORST_NODE_ATTACH_ONLY"
)

// EnvNodeAttachOnly is FORST_NODE_ATTACH_ONLY; see block comment above.
const EnvNodeAttachOnly = envNodeAttachOnly

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

func stripNodeOptionImports(existing string, substrings ...string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return ""
	}
	parts := strings.Fields(existing)
	var kept []string
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == "--import" && i+1 < len(parts) {
			path := parts[i+1]
			if nodeOptionImportMatches(path, substrings...) {
				i++
				continue
			}
			kept = append(kept, part, path)
			i++
			continue
		}
		if strings.HasPrefix(part, "--import=") {
			path := strings.TrimPrefix(part, "--import=")
			if nodeOptionImportMatches(path, substrings...) {
				continue
			}
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, " ")
}

func nodeOptionImportMatches(path string, substrings ...string) bool {
	for _, sub := range substrings {
		if sub != "" && strings.Contains(path, sub) {
			return true
		}
	}
	return false
}

func sanitizeHostChildEnv(env []string) []string {
	env = stripNodeOptionImportsEnv(env, "register.mjs", "register.cjs")
	return env
}

func stripNodeOptionImportsEnv(env []string, substrings ...string) []string {
	opts := lookupEnvValue(env, "NODE_OPTIONS")
	if opts == "" {
		return env
	}
	cleaned := stripNodeOptionImports(opts, substrings...)
	if cleaned == opts {
		return env
	}
	if cleaned == "" {
		return filterEnv(env, "NODE_OPTIONS")
	}
	return setEnvVar(env, "NODE_OPTIONS", cleaned)
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

func filterEnv(env []string, dropKey string) []string {
	prefix := dropKey + "="
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func setEnvDefault(env []string, key, value string) []string {
	if lookupEnvValue(env, key) != "" {
		return env
	}
	return setEnvVar(env, key, value)
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
// If the ready marker references a live process, returns an error instead of clobbering it.
func PrepareHostSocket(socketPath, readyPath string) error {
	if readyPath != "" {
		if marker, ok := readHostReadyMarker(readyPath); ok && processAlive(marker.PID) {
			return fmt.Errorf("node runtime: host already running (pid=%d, socket=%s)", marker.PID, socketPath)
		}
	}
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

func stdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func applyHostSpawnColorEnv(env []string) []string {
	if lookupEnvValue(env, "NO_COLOR") != "" {
		return env
	}
	if lookupEnvValue(env, "FORCE_COLOR") != "" {
		return env
	}
	if !stdoutIsTTY() {
		return env
	}
	return setEnvDefault(env, "FORCE_COLOR", "1")
}

func prependNodeImportArgs(shimArgs []string, importPaths ...string) []string {
	args := make([]string, 0, len(importPaths)*2+len(shimArgs))
	for _, importPath := range importPaths {
		importPath = strings.TrimSpace(importPath)
		if importPath == "" {
			continue
		}
		if strings.HasPrefix(importPath, "--import ") {
			importPath = strings.TrimSpace(strings.TrimPrefix(importPath, "--import "))
		}
		args = append(args, "--import", importPath)
	}
	return append(args, shimArgs...)
}

// portFromShimArgs extracts --port / -p from host shim argv for shims that honor PORT (e.g. remix-serve).
func portFromShimArgs(args []string) string {
	for i, arg := range args {
		switch {
		case arg == "--port" && i+1 < len(args):
			return strings.TrimSpace(args[i+1])
		case strings.HasPrefix(arg, "--port="):
			return strings.TrimSpace(strings.TrimPrefix(arg, "--port="))
		case (arg == "-p" || arg == "-P") && i+1 < len(args):
			return strings.TrimSpace(args[i+1])
		}
	}
	return ""
}

func sameResolvedExecutable(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	evalA, errA := filepath.EvalSymlinks(absA)
	evalB, errB := filepath.EvalSymlinks(absB)
	if errA != nil {
		evalA = absA
	}
	if errB != nil {
		evalB = absB
	}
	return evalA == evalB
}

// hostSpawnExecutableAndArgs returns argv for host mode. --import flags are Node
// flags; when ftconfig node.binary is a shim (e.g. remix-serve), spawn the real
// node interpreter with the shim script as the first positional argument.
func hostSpawnExecutableAndArgs(boundaryRoot, shimExecutable string, shimArgs, importPaths []string) (string, []string, error) {
	nodeBin, err := ResolveNodeBinary(boundaryRoot, "node")
	if err != nil {
		return "", nil, err
	}
	args := append([]string(nil), shimArgs...)
	executable := shimExecutable
	if !sameResolvedExecutable(shimExecutable, nodeBin) {
		executable = nodeBin
		args = append([]string{shimExecutable}, args...)
	}
	return executable, prependNodeImportArgs(args, importPaths...), nil
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
		env = setEnvVar(env, envNodeHostLeader, "1")
		env = setEnvDefault(env, "HOST", "127.0.0.1")
		if in.SocketPath != "" {
			env = setEnvVar(env, envNodeSocket, in.SocketPath)
		}
		if in.ReadyPath != "" {
			env = setEnvVar(env, envNodeHostReady, in.ReadyPath)
		}
		env = sanitizeHostChildEnv(env)
		env = applyHostSpawnColorEnv(env)
	}
	return env
}

// BuildHostSpawnCommand builds argv/env for app shim spawn in host mode.
func BuildHostSpawnCommand(in HostSpawnInput) (HostSpawnCommand, error) {
	if len(in.ShimArgs) == 0 {
		return HostSpawnCommand{}, fmt.Errorf("node runtime: hostMode requires non-empty node.args")
	}
	shimExecutable, err := ResolveNodeBinary(in.BoundaryRoot, in.Executable)
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
	var importPaths []string
	switch loader {
	case "tsx":
		tsxLoader, err := ResolveTsxLoaderPath(in.BoundaryRoot, in.WorkDir)
		if err != nil {
			return HostSpawnCommand{}, err
		}
		importPaths = append(importPaths, tsxLoader)
	default:
		return HostSpawnCommand{}, fmt.Errorf("unsupported node loader %q", loader)
	}

	childEnv := append([]string(nil), in.Env...)
	if in.HostAutoRegister {
		registerPath, err := ResolveHostRegisterPath(in.BoundaryRoot)
		if err != nil {
			return HostSpawnCommand{}, err
		}
		importPaths = append(importPaths, registerPath)
	}
	if in.HostAppReadyModule != "" {
		childEnv = setEnvVar(childEnv, envNodeAppReadyModule, in.HostAppReadyModule)
	}
	if port := portFromShimArgs(in.ShimArgs); port != "" {
		childEnv = setEnvVar(childEnv, "PORT", port)
	}
	// App shims (e.g. remix-serve) honor HOST for bind address. Force loopback unless
	// ftconfig/explicit spawn env already set HOST — parent CI env must not leak in.
	if lookupEnvValue(childEnv, "HOST") == "" {
		childEnv = setEnvVar(childEnv, "HOST", "127.0.0.1")
	}

	env := buildSpawnEnv(spawnEnvInput{
		BoundaryRoot: in.BoundaryRoot,
		FilesExclude: in.FilesExclude,
		Env:          childEnv,
		HostMode:     true,
		SocketPath:   socketPath,
		ReadyPath:    readyPath,
	})

	executable, args, err := hostSpawnExecutableAndArgs(in.BoundaryRoot, shimExecutable, in.ShimArgs, importPaths)
	if err != nil {
		return HostSpawnCommand{}, err
	}

	return HostSpawnCommand{
		Executable: executable,
		Args:       args,
		Env:        env,
		SocketPath: socketPath,
		ReadyPath:  readyPath,
	}, nil
}
