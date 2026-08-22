package nodert

import (
	"fmt"
	"path/filepath"
	"strings"

	"forst/internal/ftconfig"
)

// SpawnHookInput configures host-adaptive argv for bridge spawn.
type SpawnHookInput struct {
	BoundaryRoot       string
	Bridge             ftconfig.JSBridge
	ConfiguredBinary   string
	ModuleIDs          []string
	HostAutoRegister   bool
	ParentEnv          []string
	SearchDirs         []string // for tsx resolution
	SocketPath         string
	ReadyPath          string
}

// SpawnHooks is the resolved interpreter and argv prefix for bridge spawn.
type SpawnHooks struct {
	Interpreter string
	PrefixArgs  []string
	ExtraEnv    []string
}

func spawnHooks(in SpawnHookInput) (SpawnHooks, error) {
	interpreter, err := resolveHostInterpreter(in.BoundaryRoot, in.Bridge.Host, in.ConfiguredBinary)
	if err != nil {
		return SpawnHooks{}, err
	}

	var prefix []string
	switch in.Bridge.Host {
	case ftconfig.JSHostNode:
		if ftconfig.NeedTsx(in.Bridge, in.ModuleIDs) {
			search := in.SearchDirs
			if len(search) == 0 {
				search = []string{in.BoundaryRoot, in.BoundaryRoot}
			}
			tsx, err := ResolveTsxLoaderPath(search...)
			if err != nil {
				return SpawnHooks{}, err
			}
			prefix = append(prefix, "--import", tsx)
		}
		if in.HostAutoRegister {
			reg, err := ResolveHostRegisterPath(in.BoundaryRoot)
			if err != nil {
				return SpawnHooks{}, err
			}
			prefix = append(prefix, "--import", reg)
		}
	case ftconfig.JSHostBun:
		if in.HostAutoRegister {
			reg, err := ResolveHostRegisterPath(in.BoundaryRoot)
			if err != nil {
				return SpawnHooks{}, err
			}
			prefix = append(prefix, "--preload", reg)
		}
	case ftconfig.JSHostDeno:
		allowRead := denoAllowReadPaths(in.BoundaryRoot, in.SearchDirs...)
		allowWrite := denoAllowWritePaths(in.BoundaryRoot, in.SocketPath, in.ReadyPath)
		prefix = append(prefix, denoRunFlags(allowRead, allowWrite)...)
		if in.HostAutoRegister {
			reg, err := ResolveHostRegisterPath(in.BoundaryRoot)
			if err != nil {
				return SpawnHooks{}, err
			}
			prefix = append(prefix, "--preload="+reg)
		}
	default:
		return SpawnHooks{}, fmt.Errorf("node runtime: unsupported javascript.host %q", in.Bridge.Host)
	}

	env := sanitizeNodeOptionsForHost(in.ParentEnv, in.Bridge.Host)
	return SpawnHooks{
		Interpreter: interpreter,
		PrefixArgs:  prefix,
		ExtraEnv:    env,
	}, nil
}

func denoAllowReadPaths(boundaryRoot string, searchDirs ...string) string {
	seen := make(map[string]struct{})
	var paths []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = filepath.Clean(p)
		}
		if _, ok := seen[abs]; ok {
			return
		}
		seen[abs] = struct{}{}
		paths = append(paths, abs)
	}
	add(boundaryRoot)
	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}
		add(dir)
		if ext := filepath.Ext(dir); ext != "" {
			dir = filepath.Dir(dir)
			add(dir)
		}
		for d := dir; ; d = filepath.Dir(d) {
			add(d)
			add(filepath.Join(d, "node_modules"))
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
		}
	}
	return strings.Join(paths, ",")
}

func denoAllowWritePaths(boundaryRoot, socketPath, readyPath string) string {
	seen := make(map[string]struct{})
	var paths []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if strings.HasPrefix(p, "tcp://") {
			return
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = filepath.Clean(p)
		}
		dir := abs
		if ext := filepath.Ext(abs); ext != "" {
			dir = filepath.Dir(abs)
		}
		if _, ok := seen[dir]; ok {
			return
		}
		seen[dir] = struct{}{}
		paths = append(paths, dir)
	}
	add(filepath.Join(boundaryRoot, ".forst"))
	add(socketPath)
	add(readyPath)
	return strings.Join(paths, ",")
}

func denoRunFlags(allowRead, allowWrite string) []string {
	if allowWrite == "" {
		allowWrite = allowRead
	}
	return []string{
		"run",
		"--unstable-detect-cjs",
		"--allow-read=" + allowRead,
		"--allow-write=" + allowWrite,
		"--allow-env",
		"--allow-net=127.0.0.1",
	}
}

func resolveHostInterpreter(boundaryRoot string, host ftconfig.JSHost, configured string) (string, error) {
	name := string(host)
	if configured != "" {
		inferred := ftconfig.InferHostFromBinary(configured)
		base := strings.ToLower(filepath.Base(strings.TrimSpace(configured)))
		base = strings.TrimSuffix(base, ".exe")
		if inferred == host && (base == name || isBareExecutableName(configured)) {
			return ResolveNodeBinary(boundaryRoot, configured)
		}
	}
	return ResolveNodeBinary(boundaryRoot, name)
}

func sanitizeNodeOptionsForHost(env []string, host ftconfig.JSHost) []string {
	if host == ftconfig.JSHostNode {
		return env
	}
	return stripNodeOptionImportsEnv(env, "tsx")
}

func prependSpawnPrefixArgs(args []string, prefix []string) []string {
	if len(prefix) == 0 {
		return args
	}
	out := make([]string, 0, len(prefix)+len(args))
	out = append(out, prefix...)
	out = append(out, args...)
	return out
}

func hostSpawnExecutableAndArgsWithHooks(interpreter, shimExecutable string, shimArgs, prefixArgs []string) (string, []string, error) {
	args := append([]string(nil), shimArgs...)
	executable := shimExecutable
	if !sameResolvedExecutable(shimExecutable, interpreter) {
		executable = interpreter
		args = append([]string{shimExecutable}, args...)
	}
	return executable, prependSpawnPrefixArgs(args, prefixArgs), nil
}

func mergeEnv(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	out := append([]string(nil), base...)
	for _, e := range extra {
		if i := strings.IndexByte(e, '='); i > 0 {
			key := e[:i]
			out = filterEnv(out, key)
		}
		out = append(out, e)
	}
	return out
}
