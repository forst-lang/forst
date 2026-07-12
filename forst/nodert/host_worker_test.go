package nodert

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestHostMode_workerEnvWithoutRegisterImportDoesNotBindSocket(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	if runtime.GOOS == "windows" {
		t.Skip("host integration uses unix sockets")
	}

	root, _ := setupHostIntegrationRoot(t, hostServerWithSingleton)
	manifest := hostCounterManifest(root)
	socketPath := shortHostSocketPath(t)
	readyPath := socketPath + ".ready"

	resetSupervisorForTest()
	t.Setenv(envNodeBootstrap, "")
	t.Setenv(envNodeBinary, "")

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}

	if _, err := GetClient(); err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	t.Cleanup(func() {
		_ = Shutdown()
	})

	markerBefore, ok := readHostReadyMarker(readyPath)
	if !ok {
		t.Fatal("ready marker missing before worker spawn")
	}

	hostCmd, err := BuildHostSpawnCommand(HostSpawnInput{
		BoundaryRoot:     root,
		Executable:       "node",
		ShimArgs:         []string{"app/server.mjs"},
		WorkDir:          root,
		Loader:           "tsx",
		SocketPath:       socketPath,
		ReadyPath:        readyPath,
		HostAutoRegister: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	nodeOpts := lookupEnvValue(hostCmd.Env, "NODE_OPTIONS")
	if strings.Contains(nodeOpts, "register.mjs") {
		t.Fatalf("register.mjs must not be in NODE_OPTIONS, got %q", nodeOpts)
	}

	worker := exec.Command("node", "-e", "")
	worker.Dir = root
	worker.Env = filterEnv(hostCmd.Env, envNodeHostLeader)
	if err := worker.Run(); err != nil {
		t.Fatalf("worker node: %v", err)
	}

	markerAfter, ok := readHostReadyMarker(readyPath)
	if !ok {
		t.Fatal("ready marker missing after worker spawn")
	}
	if !processAlive(markerAfter.PID) {
		t.Fatalf("leader pid %d not alive after worker spawn", markerAfter.PID)
	}
	if markerAfter.PID != markerBefore.PID {
		t.Fatalf("ready pid before=%d after=%d", markerBefore.PID, markerAfter.PID)
	}
}
