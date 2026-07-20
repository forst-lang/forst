package nodert

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMergeNodeOptions(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		add      []string
		want     string
	}{
		{name: "unset", existing: "", add: []string{"--import /x"}, want: "--import /x"},
		{name: "preserves_require", existing: "--require ./a.cjs", add: []string{"--import /x"}, want: "--require ./a.cjs --import /x"},
		{name: "idempotent_same_import", existing: "--import /x", add: []string{"--import /x"}, want: "--import /x"},
		{name: "appends_different_import", existing: "--import /y", add: []string{"--import /x"}, want: "--import /y --import /x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeNodeOptions(tc.existing, tc.add...)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestResolveNodeBinary_relativeShim(t *testing.T) {
	root := t.TempDir()
	shimDir := filepath.Join(root, "node_modules", ".bin")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(shimDir, "mock-node")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveNodeBinary(root, "node_modules/.bin/mock-node")
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(shim)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveNodeBinary_missingFile(t *testing.T) {
	root := t.TempDir()
	_, err := ResolveNodeBinary(root, "missing/shim")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "executable not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveHostSocketPath_rejectsAbsolute(t *testing.T) {
	_, _, err := ResolveHostSocketPath("/tmp/root", "/abs/sock")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildBootstrapSpawnCommand_argvAndEnv(t *testing.T) {
	root := t.TempDir()
	bootstrap := filepath.Join(root, "bootstrap.js")
	if err := os.WriteFile(bootstrap, []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	nodePath := filepath.Join(root, "node")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildBootstrapSpawnCommand(BootstrapSpawnInput{
		BoundaryRoot:  root,
		Executable:    nodePath,
		BootstrapPath: bootstrap,
		WorkDir:       root,
		Loader:        "tsx",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cmd.Args) != 1 || cmd.Args[0] != bootstrap {
		t.Fatalf("args = %#v", cmd.Args)
	}
	nodeOpts := lookupEnvValue(cmd.Env, "NODE_OPTIONS")
	if !strings.Contains(nodeOpts, "--import") || !strings.Contains(nodeOpts, "loader.mjs") {
		t.Fatalf("NODE_OPTIONS = %q", nodeOpts)
	}
}

func TestBuildHostSpawnCommand_requiresArgs(t *testing.T) {
	root := t.TempDir()
	_, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   "node",
		ShimArgs:     nil,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildHostSpawnCommand_autoRegisterAndAppReadyModule(t *testing.T) {
	root := t.TempDir()
	nodePath := filepath.Join(root, "shim")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	registerPath := filepath.Join(root, "node_modules", "@forst", "node-runtime", "dist", "host", "register.mjs")
	if err := os.MkdirAll(filepath.Dir(registerPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registerPath, []byte("// register"), 0o644); err != nil {
		t.Fatal(err)
	}
	appReadyModule := filepath.Join(root, "build", "server", "index.js")
	if err := os.MkdirAll(filepath.Dir(appReadyModule), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appReadyModule, []byte("// build"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot:       root,
		Executable:         nodePath,
		ShimArgs:           []string{"server.js"},
		WorkDir:            root,
		Loader:             "tsx",
		HostAutoRegister:   true,
		HostAppReadyModule: appReadyModule,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRegister := filepath.Join(root, "node_modules", "@forst", "node-runtime", "dist", "host", "register.mjs")
	wantLoader := filepath.Join(root, "node_modules", "tsx", "dist", "loader.mjs")
	nodeBin, err := ResolveNodeBinary(root, "node")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Executable != nodeBin {
		t.Fatalf("executable = %q want node %q", cmd.Executable, nodeBin)
	}
	wantShim, err := filepath.EvalSymlinks(nodePath)
	if err != nil {
		wantShim = nodePath
	}
	wantArgs := []string{"--import", wantLoader, "--import", wantRegister, wantShim, "server.js"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("args = %#v want %#v", cmd.Args, wantArgs)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q want %q (full args = %#v)", i, cmd.Args[i], wantArgs[i], cmd.Args)
		}
	}
	if lookupEnvValue(cmd.Env, envNodeHost) != "1" {
		t.Fatalf("env = %#v", cmd.Env)
	}
	if lookupEnvValue(cmd.Env, envNodeHostLeader) != "1" {
		t.Fatalf("missing leader env: %#v", cmd.Env)
	}
	if lookupEnvValue(cmd.Env, envNodeSocket) == "" {
		t.Fatalf("missing socket env: %#v", cmd.Env)
	}
	nodeOpts := lookupEnvValue(cmd.Env, "NODE_OPTIONS")
	if strings.Contains(nodeOpts, "register.mjs") {
		t.Fatalf("register.mjs must not be in NODE_OPTIONS, got %q", nodeOpts)
	}
	if got := lookupEnvValue(cmd.Env, envNodeAppReadyModule); got != appReadyModule {
		t.Fatalf("FORST_NODE_APP_READY_MODULE = %q want %q", got, appReadyModule)
	}
}

func TestBuildHostSpawnCommand_setsHostEnv(t *testing.T) {
	root := t.TempDir()
	nodePath := filepath.Join(root, "shim")
	if err := os.WriteFile(nodePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   nodePath,
		ShimArgs:     []string{"server.js"},
		WorkDir:      root,
		Loader:       "tsx",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantLoader := filepath.Join(root, "node_modules", "tsx", "dist", "loader.mjs")
	nodeBin, err := ResolveNodeBinary(root, "node")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Executable != nodeBin {
		t.Fatalf("executable = %q want node %q", cmd.Executable, nodeBin)
	}
	wantShim, err := filepath.EvalSymlinks(nodePath)
	if err != nil {
		wantShim = nodePath
	}
	wantArgs := []string{"--import", wantLoader, wantShim, "server.js"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("args = %#v want %#v", cmd.Args, wantArgs)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q want %q (full args = %#v)", i, cmd.Args[i], wantArgs[i], cmd.Args)
		}
	}
	if lookupEnvValue(cmd.Env, envNodeHost) != "1" {
		t.Fatalf("env = %#v", cmd.Env)
	}
	if lookupEnvValue(cmd.Env, envNodeHostLeader) != "1" {
		t.Fatalf("missing leader env: %#v", cmd.Env)
	}
	if lookupEnvValue(cmd.Env, "HOST") != "127.0.0.1" {
		t.Fatalf("HOST = %q want 127.0.0.1", lookupEnvValue(cmd.Env, "HOST"))
	}
	if lookupEnvValue(cmd.Env, envNodeSocket) == "" {
		t.Fatalf("missing socket env: %#v", cmd.Env)
	}
}

func TestBuildSpawnEnv_defaultsHostWhenUnset(t *testing.T) {
	env := buildSpawnEnv(spawnEnvInput{HostMode: true})
	if lookupEnvValue(env, "HOST") != "127.0.0.1" {
		t.Fatalf("HOST = %q want 127.0.0.1", lookupEnvValue(env, "HOST"))
	}
}

func TestStripNodeOptionImports_removesRegister(t *testing.T) {
	existing := "--require ./a.cjs --import /path/to/register.mjs --import /tsx/loader.mjs"
	got := stripNodeOptionImports(existing, "register.mjs")
	want := "--require ./a.cjs --import /tsx/loader.mjs"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSanitizeHostChildEnv_stripsRegisterFromInheritedNodeOptions(t *testing.T) {
	env := sanitizeHostChildEnv([]string{
		"NODE_OPTIONS=--import /app/register.mjs --max-old-space-size=4096",
		"PATH=/usr/bin",
	})
	nodeOpts := lookupEnvValue(env, "NODE_OPTIONS")
	if strings.Contains(nodeOpts, "register.mjs") {
		t.Fatalf("NODE_OPTIONS = %q", nodeOpts)
	}
	if !strings.Contains(nodeOpts, "max-old-space-size=4096") {
		t.Fatalf("NODE_OPTIONS = %q", nodeOpts)
	}
}

func TestBuildSpawnEnv_preservesExplicitHost(t *testing.T) {
	env := buildSpawnEnv(spawnEnvInput{
		HostMode: true,
		Env:      []string{"HOST=0.0.0.0"},
	})
	if lookupEnvValue(env, "HOST") != "0.0.0.0" {
		t.Fatalf("HOST = %q want 0.0.0.0", lookupEnvValue(env, "HOST"))
	}
}

func TestApplyHostSpawnColorEnv_respectsNoColor(t *testing.T) {
	env := applyHostSpawnColorEnv([]string{"NO_COLOR=1"})
	if lookupEnvValue(env, "FORCE_COLOR") != "" {
		t.Fatalf("FORCE_COLOR = %q want unset", lookupEnvValue(env, "FORCE_COLOR"))
	}
}

func TestApplyHostSpawnColorEnv_preservesExistingForceColor(t *testing.T) {
	env := applyHostSpawnColorEnv([]string{"FORCE_COLOR=2"})
	if lookupEnvValue(env, "FORCE_COLOR") != "2" {
		t.Fatalf("FORCE_COLOR = %q want 2", lookupEnvValue(env, "FORCE_COLOR"))
	}
}

func TestPrependNodeImportArgs(t *testing.T) {
	got := prependNodeImportArgs([]string{"server.js"}, "/tsx/loader.mjs", "/host/register.mjs")
	want := []string{"--import", "/tsx/loader.mjs", "--import", "/host/register.mjs", "server.js"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %#v want %#v", got, want)
		}
	}
}

func TestBuildHostSpawnCommand_directNodeBinary(t *testing.T) {
	root := t.TempDir()
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodeBin, err := ResolveNodeBinary(root, "node")
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   "node",
		ShimArgs:     []string{"server.js"},
		WorkDir:      root,
		Loader:       "tsx",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Executable != nodeBin {
		t.Fatalf("executable = %q want %q", cmd.Executable, nodeBin)
	}
	wantLoader := filepath.Join(root, "node_modules", "tsx", "dist", "loader.mjs")
	wantArgs := []string{"--import", wantLoader, "server.js"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("args = %#v want %#v", cmd.Args, wantArgs)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q want %q", i, cmd.Args[i], wantArgs[i])
		}
	}
}

func TestBuildHostSpawnCommand_nonNodeShimUsesNodeInterpreter(t *testing.T) {
	root := t.TempDir()
	shim := filepath.Join(root, "node_modules", ".bin", "remix-serve")
	if err := os.MkdirAll(filepath.Dir(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shim, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodeBin, err := ResolveNodeBinary(root, "node")
	if err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   "node_modules/.bin/remix-serve",
		ShimArgs:     []string{"build/server/index.js", "--port", "3000"},
		WorkDir:      root,
		Loader:       "tsx",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Executable != nodeBin {
		t.Fatalf("executable = %q want node %q", cmd.Executable, nodeBin)
	}
	wantLoader := filepath.Join(root, "node_modules", "tsx", "dist", "loader.mjs")
	wantShim, err := filepath.EvalSymlinks(shim)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"--import", wantLoader, wantShim, "build/server/index.js", "--port", "3000"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("args = %#v want %#v", cmd.Args, wantArgs)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q want %q (full args = %#v)", i, cmd.Args[i], wantArgs[i], cmd.Args)
		}
	}
	assertEnvVar(t, cmd.Env, "PORT", "3000")
	assertEnvVar(t, cmd.Env, "HOST", "127.0.0.1")
}

func TestBuildHostSpawnCommand_overridesInheritedHostFromParentEnv(t *testing.T) {
	t.Setenv("HOST", "runnerspecific")
	root := t.TempDir()
	shim := filepath.Join(root, "node_modules", ".bin", "remix-serve")
	if err := os.MkdirAll(filepath.Dir(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shim, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   "node_modules/.bin/remix-serve",
		ShimArgs:     []string{"build/server/index.js", "--port", "6322"},
		WorkDir:      root,
		Loader:       "tsx",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertEnvVar(t, cmd.Env, "HOST", "127.0.0.1")
	assertEnvVar(t, cmd.Env, "PORT", "6322")
}

func TestBuildHostSpawnCommand_preservesExplicitHostInSpawnEnv(t *testing.T) {
	t.Setenv("HOST", "runnerspecific")
	root := t.TempDir()
	shim := filepath.Join(root, "node_modules", ".bin", "remix-serve")
	if err := os.MkdirAll(filepath.Dir(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shim, []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tsxDir := filepath.Join(root, "node_modules", "tsx", "dist")
	if err := os.MkdirAll(tsxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tsxDir, "loader.mjs"), []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot: root,
		Executable:   "node_modules/.bin/remix-serve",
		ShimArgs:     []string{"build/server/index.js", "--port", "6322"},
		WorkDir:      root,
		Loader:       "tsx",
		Env:          []string{"HOST=0.0.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertEnvVar(t, cmd.Env, "HOST", "0.0.0.0")
}

func TestReadyPathForSocket_emptyReturnsEmpty(t *testing.T) {
	if got := readyPathForSocket(""); got != "" {
		t.Fatalf("readyPathForSocket(\"\") = %q want empty", got)
	}
}

func TestEnsureUnixSocketPathLength_truncatesLongPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket path shortening")
	}
	long := filepath.Join("/tmp", strings.Repeat("a", 120), "node-bootstrap.sock")
	if len(long) <= maxUnixSocketPathLen {
		t.Fatalf("test path too short: len=%d", len(long))
	}

	got := ensureUnixSocketPathLength(long)
	wantPrefix := filepath.Join("/tmp", "forst-bs-")
	if !strings.HasPrefix(got, wantPrefix) || !strings.HasSuffix(got, ".sock") {
		t.Fatalf("got %q want prefix %q and .sock suffix", got, wantPrefix)
	}
	if len(got) > maxUnixSocketPathLen {
		t.Fatalf("truncated path still too long: len=%d path=%q", len(got), got)
	}
	if ensureUnixSocketPathLength(long) != got {
		t.Fatal("expected deterministic shortening")
	}
	other := ensureUnixSocketPathLength(long + "x")
	if other == got {
		t.Fatalf("different inputs should not collide: %q", got)
	}
}

func TestPrepareHostSocket_rejectsLiveHost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket test")
	}
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "node.sock")
	readyPath := socketPath + ".ready"
	marker := fmt.Sprintf(`{"pid":%d,"socket":%q,"phase":"app"}`+"\n", os.Getpid(), socketPath)
	if err := os.WriteFile(readyPath, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	err := PrepareHostSocket(socketPath, readyPath)
	if err == nil {
		t.Fatal("expected error for live host")
	}
	if !strings.Contains(err.Error(), "host already running") {
		t.Fatalf("err = %v", err)
	}
}
