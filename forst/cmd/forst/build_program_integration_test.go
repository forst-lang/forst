package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"forst/internal/programbuild"
)

func TestBuiltProgramBinary_healthAndInvoke_withoutGoOnPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping built program integration in short mode")
	}
	fixture := writeEmbeddedBuildFixture(t, embeddedBuildFixtureOpts{
		ftconfig: `{
  "server": {"embedded": true, "port": "6399"},
  "files": {"include": ["**/*.ft"]}
}`,
		mainFT: `package main

type EchoRequest = {
	message: String
}

type EchoResponse = {
	echo: String,
	timestamp: Int
}

func Echo(input EchoRequest) {
	return {
		echo: input.message,
		timestamp: 42
	}
}

func main() {
	println("embedded invoke listening")
}
`,
	})
	outDir := filepath.Join(fixture.dir, "program-build")
	manifest := buildProgram(t, fixture.c, outDir)
	if manifest.Kind != programbuild.KindProgram {
		t.Fatalf("manifest kind = %q", manifest.Kind)
	}

	binPath := filepath.Join(outDir, manifest.Binary)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, binPath)
	cmd.Dir = fixture.dir
	cmd.Env = []string{
		"FORST_BOUNDARY_ROOT=" + fixture.dir,
		"FORST_INVOKE_TRANSPORT=tcp",
		"FORST_INVOKE_AUTH=off",
		"FORST_INVOKE_PORT=6399",
		"PATH=/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start binary: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
	})

	baseURL := "http://127.0.0.1:6399"
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health status = %d", resp.StatusCode)
	}

	invokeBody := strings.NewReader(`{"package":"main","function":"Echo","args":[{"message":"hello"}]}`)
	invokeResp, err := http.Post(baseURL+"/invoke", "application/json", invokeBody)
	if err != nil {
		t.Fatalf("POST /invoke: %v", err)
	}
	defer func() { _ = invokeResp.Body.Close() }()
	if invokeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(invokeResp.Body)
		t.Fatalf("POST /invoke status = %d body = %s", invokeResp.StatusCode, body)
	}
}
