package bridgert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"forst/internal/ftconfig"
	"forst/internal/unixpath"
)

const nodeSocketTmpPrefix = "forst-bs-"

func ensureUnixSocketPathLength(abs string) string {
	return unixpath.EnsureLength(abs, nodeSocketTmpPrefix)
}

func readyPathForSocket(socketPath string) string {
	if socketPath == "" {
		return ""
	}
	return socketPath + ".ready"
}

// Host-mode environment variable names.
//
// Go sets FORST_BRIDGE_HOST, FORST_BRIDGE_HOST_LEADER, FORST_BRIDGE_SOCKET, and
// FORST_BRIDGE_HOST_READY on the direct app-shim child when ftconfig bridge.hostMode
// is true (BuildHostSpawnCommand → buildSpawnEnv). @forst/runtime reads them
// in host/register.mjs and host.ts. Child processes spawned by the shim (Vite
// workers, cluster forks, etc.) may inherit the values but must not act as host
// leaders — see FORST_BRIDGE_HOST_LEADER and register preload via argv, not
// NODE_OPTIONS. FORST_BRIDGE_ATTACH_ONLY is separate: forst dev sets it on the
// parent process for watch-reload go run children (see below).
//
//   FORST_BRIDGE_HOST
//     Gate for host RPC in Node. When "1", startForstNodeHost() may bind the RPC
//     socket; when unset, host code no-ops. Set only in host mode; bootstrap mode
//     does not set this variable (bootstrap listens unconditionally via bootstrap.js).
//
//   FORST_BRIDGE_HOST_LEADER
//     Marks the process Go spawned as the sole host leader. startForstNodeHost()
//     requires "1" and register.mjs in process.execArgv; workers that inherit
//     FORST_BRIDGE_HOST without leader/preload skip binding (host_skip_non_leader).
//     Do not set manually except for tests.
//
//   FORST_BRIDGE_SOCKET
//     Absolute path to the Unix domain socket (loopback TCP URL on Windows) where
//     Node listens for Go RPC. Host defaults from bridge.hostSocket under the boundary
//     root (.forst/bridge.sock); bootstrap defaults to .forst/node-bootstrap.sock.
//     Go dials this after readiness; may also be read at spawn planning time via
//     ResolveHostSocketPath / ResolveBootstrapSocketPath when set in the parent environment.
//
//   FORST_BRIDGE_HOST_READY
//     Absolute path to a JSON readiness marker (typically socketPath + ".ready").
//     The Node process writes {"pid", "socket", "phase"} after listen and/or app init;
//     Go polls until phase is "app" before connecting. Bootstrap writes phase "app"
//     immediately after listen; host may defer until app shims finish bootstrapping.
//
//   FORST_BRIDGE_ATTACH_ONLY
//     Attach-only gate for Go-side host supervision. When "1", bridgert dials an
//     existing host via FORST_BRIDGE_SOCKET / FORST_BRIDGE_HOST_READY but never
//     spawns a new shim. forst dev sets this on the parent after EnsureRunning
//     so each watch-reload go run child inherits it; one-shot forst run leaves
//     it unset so the binary may spawn the host on first GetClient(). If no
//     live host is reachable, GetClient fails with an attach-only error instead
//     of starting a second Vite process.
const (
	envBridgeHost       = "FORST_BRIDGE_HOST"
	envBridgeHostLeader = "FORST_BRIDGE_HOST_LEADER"
	envBridgeSocket     = "FORST_BRIDGE_SOCKET"
	envBridgeHostReady  = "FORST_BRIDGE_HOST_READY"
	envBridgeAttachOnly = "FORST_BRIDGE_ATTACH_ONLY"
)

// EnvBridgeAttachOnly is FORST_BRIDGE_ATTACH_ONLY.
const EnvBridgeAttachOnly = envBridgeAttachOnly

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

// ResolveBridgeBinary returns the Node or shim executable path.
// Priority: FORST_BRIDGE_BINARY env → configured → "node".
func ResolveBridgeBinary(boundaryRoot, configured string) (string, error) {
	candidate := configured
	if path := os.Getenv(envBridgeBinary); path != "" {
		candidate = path
	}
	if candidate == "" {
		candidate = "node"
	}
	return resolveExecutablePath(boundaryRoot, candidate)
}

func resolveExecutablePath(boundaryRoot, candidate string) (string, error) {
	if candidate == "" {
		return "", bridgeRuntimeErr("executable path is empty")
	}

	resolved := candidate
	if !filepath.IsAbs(resolved) && !isBareExecutableName(resolved) {
		if boundaryRoot == "" {
			var err error
			boundaryRoot, err = os.Getwd()
			if err != nil {
				return "", bridgeRuntimeErr("getwd: %w", err)
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
		return "", bridgeRuntimeErr("resolve executable path: %w", err)
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err != nil {
		eval = abs
	}
	st, err := os.Stat(eval)
	if err != nil {
		return "", bridgeRuntimeErr("executable not found at %s: %w", eval, err)
	}
	if st.IsDir() {
		return "", bridgeRuntimeErr("executable path is a directory: %s", eval)
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
		return "", bridgeRuntimeErr("executable not found at %s", name)
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
	if path := strings.TrimSpace(os.Getenv(envBridgeSocket)); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", "", bridgeRuntimeErr("resolve host socket: %w", err)
		}
		abs = ensureUnixSocketPathLength(abs)
		return abs, readyPathForSocket(abs), nil
	}
	if configured == "" {
		configured = ".forst/bridge.sock"
	}
	if filepath.IsAbs(configured) {
		return "", "", bridgeRuntimeErr("hostSocket must be relative to boundary root")
	}
	clean := filepath.Clean(configured)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || strings.Contains(clean, "..") {
		return "", "", bridgeRuntimeErr("hostSocket escapes boundary: %q", configured)
	}
	if boundaryRoot == "" {
		var err error
		boundaryRoot, err = os.Getwd()
		if err != nil {
			return "", "", bridgeRuntimeErr("getwd: %w", err)
		}
	}
	socketPath, err := filepath.Abs(filepath.Join(boundaryRoot, clean))
	if err != nil {
		return "", "", bridgeRuntimeErr("resolve host socket: %w", err)
	}
	socketPath = ensureUnixSocketPathLength(socketPath)
	return socketPath, readyPathForSocket(socketPath), nil
}

// ResolveBootstrapSocketPath returns the absolute Unix socket path for bootstrap mode.
func ResolveBootstrapSocketPath(boundaryRoot string) (string, string, error) {
	return ResolveHostSocketPath(boundaryRoot, ".forst/node-bootstrap.sock")
}

// PrepareHostSocket removes stale socket and ready marker files.
// If the ready marker references a live process, returns an error instead of clobbering it.
func PrepareHostSocket(socketPath, readyPath string) error {
	if readyPath != "" {
		if marker, ok := readHostReadyMarker(readyPath); ok && processAlive(marker.PID) {
			return bridgeRuntimeErr("host already running (pid=%d, socket=%s)", marker.PID, socketPath)
		}
	}
	if socketPath != "" {
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return bridgeRuntimeErr("remove stale host socket: %w", err)
		}
	}
	if readyPath != "" {
		if err := os.Remove(readyPath); err != nil && !os.IsNotExist(err) {
			return bridgeRuntimeErr("remove stale host ready file: %w", err)
		}
	}
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return bridgeRuntimeErr("create host socket dir: %w", err)
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

// hostSpawnExecutableAndArgs returns argv for host mode using the configured bridge interpreter.
func hostSpawnExecutableAndArgs(boundaryRoot, interpreter, shimExecutable string, shimArgs, prefixArgs []string) (string, []string, error) {
	if interpreter == "" {
		var err error
		interpreter, err = ResolveBridgeBinary(boundaryRoot, "node")
		if err != nil {
			return "", nil, err
		}
	}
	return hostSpawnExecutableAndArgsWithHooks(interpreter, shimExecutable, shimArgs, prefixArgs)
}

// BootstrapSpawnInput configures bootstrap-mode child spawn.
type BootstrapSpawnInput struct {
	BoundaryRoot  string
	ModulesDir    string
	Executable    string
	BootstrapPath string
	WorkDir       string
	Bridge        ftconfig.Bridge
	ModuleIDs     []string
	SocketPath    string
	ReadyPath     string
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

	hooks, err := spawnHooks(SpawnHookInput{
		BoundaryRoot:     in.BoundaryRoot,
		ModulesDir:       in.ModulesDir,
		Bridge:           in.Bridge,
		ConfiguredBinary: in.Executable,
		ModuleIDs:        in.ModuleIDs,
		HostAutoRegister: false,
		ParentEnv:        in.Env,
		SearchDirs:       []string{bootstrapPath, in.WorkDir},
		SocketPath:       in.SocketPath,
		ReadyPath:        in.ReadyPath,
	})
	if err != nil {
		return BootstrapSpawnCommand{}, err
	}

	var nodeOpts []string
	// tsx for bootstrap is passed via argv prefix, not NODE_OPTIONS

	env := buildSpawnEnv(spawnEnvInput{
		BoundaryRoot: in.BoundaryRoot,
		ModulesDir:   in.ModulesDir,
		FilesExclude: in.FilesExclude,
		Env:          mergeEnv(in.Env, hooks.ExtraEnv),
		NodeOptions:  nodeOpts,
		SocketPath:   in.SocketPath,
		ReadyPath:    in.ReadyPath,
	})

	args := append([]string(nil), hooks.PrefixArgs...)
	args = append(args, bootstrapPath)
	args = append(args, in.ExtraArgs...)
	return BootstrapSpawnCommand{
		Executable: hooks.Interpreter,
		Args:       args,
		Env:        env,
	}, nil
}

// HostSpawnInput configures host-mode app shim spawn.
type HostSpawnInput struct {
	BoundaryRoot       string
	ModulesDir         string
	Executable         string
	ShimArgs           []string
	WorkDir            string
	Bridge             ftconfig.Bridge
	ModuleIDs          []string
	SocketPath         string
	ReadyPath          string
	FilesExclude       []string
	Env                []string
	HostAutoRegister   bool
	HostAppReadyModule string
	AuthRelay          *HostInvokeAuthRelay
}

// HostSpawnCommand is the resolved host-mode spawn invocation.
type HostSpawnCommand struct {
	Executable string
	Args       []string
	Env        []string
	SocketPath string
	ReadyPath  string
	ExtraFiles []*os.File
}

type spawnEnvInput struct {
	BoundaryRoot string
	ModulesDir   string
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
		ModulesDir:   in.ModulesDir,
		FilesExclude: in.FilesExclude,
		Env:          in.Env,
	}
	env := buildNodeChildEnv(opts)
	existing := lookupEnvValue(env, "NODE_OPTIONS")
	merged := mergeNodeOptions(existing, in.NodeOptions...)
	env = setEnvVar(env, "NODE_OPTIONS", merged)
	if in.SocketPath != "" {
		env = setEnvVar(env, envBridgeSocket, in.SocketPath)
	}
	if in.ReadyPath != "" {
		env = setEnvVar(env, envBridgeHostReady, in.ReadyPath)
	}
	if in.HostMode {
		env = setEnvVar(env, envBridgeHost, "1")
		env = setEnvVar(env, envBridgeHostLeader, "1")
		env = setEnvDefault(env, "HOST", "127.0.0.1")
		env = sanitizeHostChildEnv(env)
		env = applyHostSpawnColorEnv(env)
	}
	return env
}

// BuildHostSpawnCommand builds argv/env for app shim spawn in host mode.
func BuildHostSpawnCommand(in HostSpawnInput) (HostSpawnCommand, error) {
	if len(in.ShimArgs) == 0 {
		return HostSpawnCommand{}, bridgeHostErr("hostMode requires non-empty bridge.args")
	}
	shimExecutable, err := ResolveBridgeBinary(in.BoundaryRoot, in.Executable)
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

	hooks, err := spawnHooks(SpawnHookInput{
		BoundaryRoot:       in.BoundaryRoot,
		Bridge:             in.Bridge,
		ConfiguredBinary:   in.Executable,
		ModuleIDs:          in.ModuleIDs,
		HostAutoRegister:   in.HostAutoRegister,
		ParentEnv:          in.Env,
		SearchDirs:         []string{in.BoundaryRoot, in.WorkDir},
		SocketPath:         socketPath,
		ReadyPath:          readyPath,
	})
	if err != nil {
		return HostSpawnCommand{}, err
	}

	childEnv := mergeEnv(in.Env, hooks.ExtraEnv)
	if in.HostAppReadyModule != "" {
		childEnv = setEnvVar(childEnv, envBridgeAppReadyModule, in.HostAppReadyModule)
	}
	if port := portFromShimArgs(in.ShimArgs); port != "" {
		childEnv = setEnvVar(childEnv, "PORT", port)
	}
	// App shims (e.g. remix-serve) honor HOST for bind address. Use explicit spawn env,
	// else parent HOST (e.g. HOST=0.0.0.0 for Docker), else loopback.
	if lookupEnvValue(childEnv, "HOST") == "" {
		if parentHost := os.Getenv("HOST"); parentHost != "" {
			childEnv = setEnvVar(childEnv, "HOST", parentHost)
		} else {
			childEnv = setEnvVar(childEnv, "HOST", "127.0.0.1")
		}
	}

	env := buildSpawnEnv(spawnEnvInput{
		BoundaryRoot: in.BoundaryRoot,
		ModulesDir:   in.ModulesDir,
		FilesExclude: in.FilesExclude,
		Env:          childEnv,
		HostMode:     true,
		SocketPath:   socketPath,
		ReadyPath:    readyPath,
	})

	executable, args, err := hostSpawnExecutableAndArgs(in.BoundaryRoot, hooks.Interpreter, shimExecutable, in.ShimArgs, hooks.PrefixArgs)
	if err != nil {
		return HostSpawnCommand{}, err
	}

	var extraFiles []*os.File
	if in.AuthRelay != nil && SupportsInvokeAuthFDHandoff() {
		env = setEnvVar(env, EnvInvokeAuthRecvFD, fmt.Sprintf("%d", in.AuthRelay.HostRecvFD()))
		if f := in.AuthRelay.HostExtraFile(); f != nil {
			extraFiles = append(extraFiles, f)
		}
	}

	return HostSpawnCommand{
		Executable: executable,
		Args:       args,
		Env:        env,
		SocketPath: socketPath,
		ReadyPath:  readyPath,
		ExtraFiles: extraFiles,
	}, nil
}
