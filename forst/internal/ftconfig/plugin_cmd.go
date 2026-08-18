package ftconfig

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EnvForstPluginDir overrides plugin executable search with a single directory.
const EnvForstPluginDir = "FORST_PLUGIN_DIR"

// EnvForstPluginsPath adds extra plugin search directories (os.PathListSeparator-separated).
const EnvForstPluginsPath = "FORST_PLUGINS_PATH"

// OfficialPluginCommands are bare names shipped with compiler releases.
var OfficialPluginCommands = []string{
	"forst-gen-jsonschema",
	"forst-gen-orpc",
	"forst-gen-file-routes",
	"forst-gen-react-router",
	"forst-gen-echo",
}

func resolvePluginCmd(boundaryRoot, cmd string) (string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", fmt.Errorf("cmd is required")
	}
	if filepath.IsAbs(cmd) {
		return cmd, nil
	}
	if strings.Contains(cmd, string(filepath.Separator)) || strings.HasPrefix(cmd, ".") {
		return filepath.Join(filepath.Clean(boundaryRoot), cmd), nil
	}
	if path, ok := lookupPluginExecutable(cmd); ok {
		return path, nil
	}
	return "", fmt.Errorf("plugin cmd %q not found (set %s, install official plugins, or add to PATH)", cmd, EnvForstPluginDir)
}

func lookupPluginExecutable(name string) (string, bool) {
	for _, dir := range pluginSearchDirs() {
		if path, ok := executableInDir(dir, name); ok {
			return path, true
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return path, true
}

func pluginSearchDirs() []string {
	seen := map[string]struct{}{}
	var dirs []string
	add := func(dir string) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		dirs = append(dirs, clean)
	}

	if extra := os.Getenv(EnvForstPluginsPath); extra != "" {
		for _, part := range filepath.SplitList(extra) {
			add(part)
		}
	}
	add(os.Getenv(EnvForstPluginDir))

	if exe, err := os.Executable(); err == nil {
		if resolved, symErr := filepath.EvalSymlinks(exe); symErr == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		add(dir)
		add(filepath.Join(dir, "plugins"))
		add(filepath.Join(filepath.Dir(dir), "plugins"))
	}
	return dirs
}

func executableInDir(dir, name string) (string, bool) {
	candidates := []string{name}
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(name), ".exe") {
		candidates = append(candidates, name+".exe")
	}
	for _, base := range candidates {
		path := filepath.Join(dir, base)
		if isExecutableFile(path) {
			return path, true
		}
	}
	return "", false
}

func isExecutableFile(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return st.Mode()&0o111 != 0
}
