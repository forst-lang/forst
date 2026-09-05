package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"forst/internal/compiler"
	"forst/internal/invokeserver"
	"forst/internal/programbuild"
)

func TestBuiltHostModeProgram_invokeReadyHandoffAndHostSpawn(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping hostMode built program integration in short mode")
	}
	if runtime.GOOS == "windows" {
		t.Skip("hostMode auth FD handoff requires Unix")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}

	forstModule := forstCompilerModuleRoot(t)
	repoRoot := filepath.Clean(filepath.Join(forstModule, ".."))
	hostJS := filepath.Join(repoRoot, "packages", "runtime", "dist", "host.js")
	if _, err := os.Stat(hostJS); err != nil {
		t.Skipf("/runtime not built: %v", err)
	}

	dir := t.TempDir()
	legacyDir := filepath.Join(dir, "legacy")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	counterTS := `declare const globalThis: { __forstTest?: { n: number } };
export function inc(): number {
  if (!globalThis.__forstTest) {
    globalThis.__forstTest = { n: 0 };
  }
  return ++globalThis.__forstTest.n;
}
`
	if err := os.WriteFile(filepath.Join(legacyDir, "counter.ts"), []byte(counterTS), 0o644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(dir, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serverMJS := `import { signalForstAppReady } from "file://` + filepath.ToSlash(hostJS) + `";

globalThis.__forstTest = { n: 1 };
await signalForstAppReady();
`
	if err := os.WriteFile(filepath.Join(appDir, "server.mjs"), []byte(serverMJS), 0o644); err != nil {
		t.Fatal(err)
	}

	linkMonorepoNodeDeps(t, dir, repoRoot)

	nodeBin, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick invoke port: %v", err)
	}
	invokePort := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	portStr := strconv.Itoa(invokePort)
	healthURL := "http://127.0.0.1:" + portStr + "/health"

	socketDir := filepath.Join(dir, ".forst")
	if err := os.MkdirAll(socketDir, 0o750); err != nil {
		t.Fatal(err)
	}

	ftconfig := `{
  "server": {"embedded": true, "port": "` + portStr + `"},
  "files": {"include": ["**/*.ft", "**/*.ts"], "exclude": ["**/node_modules/**"]},
  "bridge": {
    "enabled": true,
    "runtimeEnabled": true,
    "hostMode": true,
    "hostAutoRegister": true,
    "binary": "` + filepath.ToSlash(nodeBin) + `",
    "hostSocket": ".forst/node.sock",
    "args": ["app/server.mjs"],
    "hostReadyTimeoutSeconds": 60
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(ftconfig), 0o644); err != nil {
		t.Fatal(err)
	}

	mainFT := `package main

import "./legacy/counter" js
import "strconv"

type EchoRequest = {
	message: String
}

type EchoResponse = {
	echo: String
}

func Echo(input EchoRequest) {
	return {
		echo: input.message
	}
}

func main() {
	first := counter.inc()
	ensure first is Ok()
	println(strconv.FormatFloat(first, 'f', 0, 64))
}
`
	entry := filepath.Join(dir, "main.ft")
	if err := os.WriteFile(entry, []byte(mainFT), 0o644); err != nil {
		t.Fatal(err)
	}

	forstGomod := filepath.Join(dir, ".forst-gomod")
	if err := os.MkdirAll(forstGomod, 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := "module example.com/hostmode\n\ngo 1.26.0\n\nrequire forst v0.0.0\n\nreplace forst => " + forstModule + "\n"
	if err := os.WriteFile(filepath.Join(forstGomod, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(dir, "program-build")
	c := compiler.New(compiler.Args{
		Command:            "build",
		FilePath:           entry,
		PackageRoot:        dir,
		ExportStructFields: true,
		LogLevel:           "error",
	}, exampleTestLogger())
	manifest := buildProgram(t, c, outDir)
	if manifest.Kind != programbuild.KindProgram {
		t.Fatalf("manifest kind = %q want %q", manifest.Kind, programbuild.KindProgram)
	}
	if !manifest.HostMode {
		t.Fatal("manifest hostMode want true")
	}
	if manifest.Binary != "bin/main" {
		t.Fatalf("manifest binary = %q want bin/main", manifest.Binary)
	}

	binPath := filepath.Join(outDir, manifest.Binary)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath)
	cmd.Dir = dir
	cmd.Env = []string{
		"FORST_ROOT=" + dir,
		"FORST_BRIDGE_BINARY=" + nodeBin,
		"FORST_INVOKE_TRANSPORT=tcp",
		"FORST_INVOKE_PORT=" + portStr,
		"PATH=/usr/bin:/bin:" + filepath.Join(dir, "node_modules", ".bin"),
		"HOME=" + os.Getenv("HOME"),
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start built program: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		if cmd.Process != nil && runtime.GOOS != "windows" {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}
		_ = cmd.Wait()
	})

	readyPath := filepath.Join(dir, ".forst", "invoke.ready")
	deadline := time.Now().Add(20 * time.Second)
	var tokenDelivery string
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(readyPath)
		if err == nil {
			var payload invokeserver.InvokeReadyPayload
			if json.Unmarshal(raw, &payload) == nil && payload.TokenDelivery != "" {
				tokenDelivery = payload.TokenDelivery
				if payload.SocketPath != "" || payload.URL != "" {
					break
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if tokenDelivery != "handoff" {
		raw, readErr := os.ReadFile(readyPath)
		t.Fatalf("invoke.ready tokenDelivery = %q want handoff (read %q err=%v)", tokenDelivery, string(raw), readErr)
	}

	hostReady := filepath.Join(dir, ".forst", "node.sock.ready")
	hostDeadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(hostDeadline) {
		if _, err := os.Stat(hostReady); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := os.Stat(hostReady); err != nil {
		t.Fatalf("host ready marker missing at %s: %v", hostReady, err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health status = %d", resp.StatusCode)
	}
}

func linkMonorepoNodeDeps(t *testing.T, root, repoRoot string) {
	t.Helper()
	tsxSrc := filepath.Join(repoRoot, "node_modules", "tsx")
	if _, err := os.Stat(tsxSrc); err != nil {
		t.Skipf("tsx not installed in monorepo: %v", err)
	}
	nodeRTSrc := filepath.Join(repoRoot, "packages", "runtime")
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(tsxSrc, filepath.Join(root, "node_modules", "tsx")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "@forst"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(nodeRTSrc, filepath.Join(root, "node_modules", "@forst", "runtime")); err != nil {
		t.Fatal(err)
	}
}
