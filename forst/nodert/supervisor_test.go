package nodert

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestGetClient_supervisorFailurePrintsStderr(t *testing.T) {
	resetSupervisorForTest()
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	ConfigureSupervisor(SupervisorConfig{
		HostMode: true,
		ShimArgs: []string{"./missing-shim.js"},
		ProcessOptions: ProcessOptions{
			BoundaryRoot: t.TempDir(),
			Loader:       "tsx",
			NodePath:     "node",
		},
	})

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	var wg sync.WaitGroup
	wg.Add(1)
	var captured bytes.Buffer
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&captured, r)
	}()

	_, clientErr := GetClient()
	_, _ = GetClient()

	_ = w.Close()
	os.Stderr = oldStderr
	wg.Wait()

	if clientErr == nil {
		t.Fatal("expected GetClient error")
	}
	out := captured.String()
	if !strings.Contains(out, "forst node runtime:") {
		t.Fatalf("stderr missing nodert prefix, got %q", out)
	}
	if !strings.Contains(out, clientErr.Error()) {
		t.Fatalf("stderr should contain supervisor error %q, got %q", clientErr, out)
	}
	if strings.Count(out, "forst node runtime:") != 1 {
		t.Fatalf("expected single stderr line, got %q", out)
	}
}
