package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"forst/internal/ftconfig"
	"forst/internal/semantic"
)

func TestRunSemanticPlugins_echo(t *testing.T) {
	root := t.TempDir()
	echoBin := buildEchoPlugin(t)

	snapshot := &semantic.GenerateRequest{
		ProtocolVersion: semantic.ProtocolVersion,
		Types: map[string]semantic.Type{
			"catalog.Username": {ID: "catalog.Username", Kind: "string"},
		},
		Functions: map[string]semantic.Function{},
	}
	plugin := ftconfig.GeneratePluginConfig{
		Name: "echo",
		Cmd:  echoBin,
		Out:  "generated/echo",
	}
	var stats generateWriteStats
	if err := runOneSemanticPlugin(root, snapshot, plugin, newGenerateLogger(), &stats); err != nil {
		t.Fatalf("runOneSemanticPlugin: %v", err)
	}
	manifest := filepath.Join(root, "generated/echo/manifest.txt")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(data), "catalog.Username") {
		t.Fatalf("manifest missing type id:\n%s", data)
	}
}

func TestRunSemanticPlugins_jsonschema(t *testing.T) {
	root := t.TempDir()
	pluginBin := buildJSONSchemaPlugin(t)

	raw, err := os.ReadFile(constraintsSnapshotGolden(t))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	var snapshot semantic.GenerateRequest
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}

	plugin := ftconfig.GeneratePluginConfig{
		Name: "jsonschema",
		Cmd:  pluginBin,
		Out:  "generated/jsonschema",
	}
	var stats generateWriteStats
	if err := runOneSemanticPlugin(root, &snapshot, plugin, newGenerateLogger(), &stats); err != nil {
		t.Fatalf("runOneSemanticPlugin: %v", err)
	}
	schemaPath := filepath.Join(root, "generated/jsonschema/schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !strings.Contains(string(data), `"minLength": 3`) {
		t.Fatalf("unexpected schema:\n%s", data)
	}
}

func TestRunSemanticPlugins_orpc(t *testing.T) {
	root := t.TempDir()
	pluginBin := buildORPCPlugin(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(repoRoot, "internal", "semantic", "testdata", "router", "snapshot.golden.json"))
	if err != nil {
		t.Fatalf("read router golden: %v", err)
	}
	var snapshot semantic.GenerateRequest
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	plugin := ftconfig.GeneratePluginConfig{Name: "orpc", Cmd: pluginBin, Out: "generated/orpc"}
	var stats generateWriteStats
	if err := runOneSemanticPlugin(root, &snapshot, plugin, newGenerateLogger(), &stats); err != nil {
		t.Fatalf("runOneSemanticPlugin: %v", err)
	}
	contract, err := os.ReadFile(filepath.Join(root, "generated/orpc/contract.ts"))
	if err != nil {
		t.Fatalf("read contract: %v", err)
	}
	if !strings.Contains(string(contract), "PlaceOrder") {
		t.Fatalf("contract missing PlaceOrder")
	}
}

func TestRunSemanticPlugins_fileRoutes(t *testing.T) {
	root := t.TempDir()
	pluginBin := buildFileRoutesPlugin(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(repoRoot, "internal", "semantic", "testdata", "layout", "snapshot.golden.json"))
	if err != nil {
		t.Fatalf("read layout golden: %v", err)
	}
	var snapshot semantic.GenerateRequest
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	plugin := ftconfig.GeneratePluginConfig{
		Name: "file-routes",
		Cmd:  pluginBin,
		Out:  "generated/api",
	}
	var stats generateWriteStats
	if err := runOneSemanticPlugin(root, &snapshot, plugin, newGenerateLogger(), &stats); err != nil {
		t.Fatalf("runOneSemanticPlugin: %v", err)
	}
	registry, err := os.ReadFile(filepath.Join(root, "generated/api/registry.ts"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	if !strings.Contains(string(registry), "/api/routes") {
		t.Fatalf("registry missing routes path:\n%s", registry)
	}
}

func TestRunSemanticPlugins_reactRouter(t *testing.T) {
	root := t.TempDir()
	pluginBin := buildReactRouterPlugin(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(repoRoot, "internal", "semantic", "testdata", "layout", "snapshot.golden.json"))
	if err != nil {
		t.Fatalf("read layout golden: %v", err)
	}
	var snapshot semantic.GenerateRequest
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	plugin := ftconfig.GeneratePluginConfig{
		Name: "rr-ssr",
		Cmd:  pluginBin,
		Out:  "generated/rr",
	}
	var stats generateWriteStats
	if err := runOneSemanticPlugin(root, &snapshot, plugin, newGenerateLogger(), &stats); err != nil {
		t.Fatalf("runOneSemanticPlugin: %v", err)
	}
	routes, err := os.ReadFile(filepath.Join(root, "generated/rr/routes.ts"))
	if err != nil {
		t.Fatalf("read routes: %v", err)
	}
	if !strings.Contains(string(routes), "forstApiRoutes") || !strings.Contains(string(routes), "api/routes") {
		t.Fatalf("routes.ts:\n%s", routes)
	}
	if _, err := os.Stat(filepath.Join(root, "generated/rr/handlers/routes.ts")); err != nil {
		t.Fatalf("missing handler: %v", err)
	}
}

func buildEchoPlugin(t *testing.T) string {
	t.Helper()
	return buildPlugin(t, "./plugins/forst-gen-echo")
}

func buildJSONSchemaPlugin(t *testing.T) string {
	t.Helper()
	return buildPlugin(t, "./plugins/forst-gen-jsonschema")
}

func buildORPCPlugin(t *testing.T) string {
	t.Helper()
	return buildPlugin(t, "./plugins/forst-gen-orpc")
}

func buildFileRoutesPlugin(t *testing.T) string {
	t.Helper()
	return buildPlugin(t, "./plugins/forst-gen-file-routes")
}

func buildReactRouterPlugin(t *testing.T) string {
	t.Helper()
	return buildPlugin(t, "./plugins/forst-gen-react-router")
}

func buildPlugin(t *testing.T, pkg string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	base := filepath.Base(pkg)
	out := filepath.Join(t.TempDir(), base)
	cmd := exec.Command("go", "build", "-o", out, pkg)
	cmd.Dir = repoRoot
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, outBytes)
	}
	return out
}

func constraintsSnapshotGolden(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(repoRoot, "internal", "semantic", "testdata", "constraints", "snapshot.golden.json")
}
