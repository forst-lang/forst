package bridgert

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"forst/internal/ftconfig"
)

const (
	compiledLegacyPaymentModuleID = "legacy/payment.js"
	typeScriptLegacyPaymentModuleID = "legacy/payment.ts"
)

const legacyPaymentAddJS = `export function add(a, b) { return { sum: a + b }; }`

const legacyPaymentAddTS = `export function add(a: number, b: number): { sum: number } {
  return { sum: a + b };
}
`

type bridgeHostE2ESpec struct {
	Host            ftconfig.BridgeHost
	Binary          string
	Format          ftconfig.LegacyModuleFormat
	ModuleID        string
	LegacySetup     func(t *testing.T, root string)
	BeforeConfigure func(t *testing.T)
}

func skipUnlessBridgeHostBinary(t *testing.T, binary string) {
	t.Helper()
	if _, err := exec.LookPath(binary); err != nil {
		if os.Getenv("FORST_REQUIRE_BRIDGE_HOST") == "1" {
			t.Fatalf("%s required but not on PATH (install or add to CI)", binary)
		}
		t.Skipf("%s not on PATH", binary)
	}
}

func skipUnlessBridgeHostTransport(t *testing.T, host ftconfig.BridgeHost) {
	t.Helper()
	if host == ftconfig.BridgeHostDeno {
		return
	}
	if runtime.GOOS == "windows" {
		t.Skip("bootstrap socket integration uses unix sockets")
	}
}

func requireBuiltBootstrap(t *testing.T) {
	t.Helper()
	bootstrap, err := ResolveBootstrapPath(repoRoot(t), "")
	if err != nil {
		t.Skipf("bootstrap not available: %v", err)
	}
	t.Setenv(envNodeBootstrap, bootstrap)
}

func writeCompiledLegacyPaymentModule(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".forst", "js", "legacy", "payment.js")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(legacyPaymentAddJS), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTypeScriptLegacyPaymentModule(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "legacy", "payment.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(legacyPaymentAddTS), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeBridgeHostFtconfig(t *testing.T, root string, host ftconfig.BridgeHost, binary string, format ftconfig.LegacyModuleFormat) {
	t.Helper()
	cfgJSON := fmt.Sprintf(`{
  "bridge": {
    "host": %q,
    "binary": %q,
    "legacyModules": { "format": %q }
  }
}`, host, binary, format)
	if err := os.WriteFile(filepath.Join(root, "ftconfig.json"), []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runBridgeHostBootstrapE2E(t *testing.T, spec bridgeHostE2ESpec) {
	t.Helper()

	if spec.Binary == "" {
		spec.Binary = string(spec.Host)
	}
	if spec.Format == "" {
		spec.Format = ftconfig.LegacyModuleCompiled
	}
	if spec.ModuleID == "" {
		if spec.Format == ftconfig.LegacyModuleTypeScript {
			spec.ModuleID = typeScriptLegacyPaymentModuleID
		} else {
			spec.ModuleID = compiledLegacyPaymentModuleID
		}
	}
	if spec.LegacySetup == nil {
		spec.LegacySetup = writeCompiledLegacyPaymentModule
	}

	skipUnlessBridgeHostBinary(t, spec.Binary)
	skipUnlessBridgeHostTransport(t, spec.Host)
	requireBuiltBootstrap(t)

	if spec.BeforeConfigure != nil {
		spec.BeforeConfigure(t)
	}

	root := t.TempDir()
	writeBridgeHostFtconfig(t, root, spec.Host, spec.Binary, spec.Format)
	spec.LegacySetup(t, root)

	manifest := Manifest{
		Version:      ManifestVersion,
		BoundaryRoot: root,
		Exports: []ExportEntry{
			{ModuleID: spec.ModuleID, Name: "add", Kind: ExportKindFunction},
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	resetSupervisorForTest()
	t.Cleanup(resetSupervisorForTest)
	if err := configureFromManifest(string(manifestJSON)); err != nil {
		t.Fatal(err)
	}
	if supervisorCfg.ProcessOptions.Bridge.Host != spec.Host {
		t.Fatalf("Bridge.Host = %q want %q", supervisorCfg.ProcessOptions.Bridge.Host, spec.Host)
	}
	if supervisorCfg.ProcessOptions.Bridge.ModuleFormat != spec.Format {
		t.Fatalf("Bridge.ModuleFormat = %q want %q", supervisorCfg.ProcessOptions.Bridge.ModuleFormat, spec.Format)
	}

	client, err := GetClient()
	if err != nil {
		t.Fatalf("GetClient: %v", err)
	}
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	type sumResult struct {
		Sum float64 `json:"sum"`
	}
	got, err := CallSync[sumResult](spec.ModuleID, "add", 40, 2)
	if err != nil {
		t.Fatalf("CallSync: %v", err)
	}
	if got.Sum != 42 {
		t.Fatalf("sum = %v want 42", got.Sum)
	}
	if err := Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
