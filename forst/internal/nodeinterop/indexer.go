package nodeinterop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"forst/internal/ftconfig"
	"forst/nodert"
)

const indexCLIFormat = "forst-index-v1"

type indexCLIOutput struct {
	Format  string    `json:"format"`
	Modules []IndexV1 `json:"modules"`
}

// RunIndexer loads forst-index-v1 data for moduleIDs under boundaryRoot by
// invoking the @forst/node-runtime indexer CLI (node dist, bun, or tsx).
func RunIndexer(boundaryRoot string, moduleIDs []string) ([]*IndexV1, error) {
	boundaryRoot = filepath.Clean(boundaryRoot)
	if boundaryRoot == "" {
		return nil, fmt.Errorf("indexer: boundaryRoot is required")
	}
	if len(moduleIDs) == 0 {
		return nil, fmt.Errorf("indexer: at least one moduleId is required")
	}

	normalized := make([]string, 0, len(moduleIDs))
	for _, moduleID := range moduleIDs {
		moduleID = filepath.ToSlash(strings.TrimSpace(moduleID))
		if moduleID == "" {
			return nil, fmt.Errorf("indexer: empty moduleId")
		}
		normalized = append(normalized, moduleID)
	}

	cliIndexes, err := runIndexerCLI(boundaryRoot, normalized)
	if err != nil {
		return nil, err
	}
	byModule := make(map[string]*IndexV1, len(cliIndexes))
	for _, idx := range cliIndexes {
		if idx != nil {
			byModule[idx.ModuleID] = idx
		}
	}
	out := make([]*IndexV1, 0, len(normalized))
	for _, moduleID := range normalized {
		idx, ok := byModule[moduleID]
		if !ok {
			return nil, fmt.Errorf("indexer: module %q not indexed", moduleID)
		}
		out = append(out, idx)
	}
	return out, nil
}

func runIndexerCLI(boundaryRoot string, moduleIDs []string) ([]*IndexV1, error) {
	cmd, err := indexerCommand(boundaryRoot, moduleIDs)
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("indexer: %s", msg)
	}

	var payload indexCLIOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("indexer: parse output: %w", err)
	}
	if payload.Format != indexCLIFormat {
		return nil, fmt.Errorf("indexer: unsupported format %q", payload.Format)
	}

	out := make([]*IndexV1, 0, len(payload.Modules))
	for i := range payload.Modules {
		idx := payload.Modules[i]
		if err := idx.Validate(); err != nil {
			return nil, err
		}
		cp := idx
		out = append(out, &cp)
	}
	return out, nil
}

func indexerCommand(boundaryRoot string, moduleIDs []string) (*exec.Cmd, error) {
	cliPath, err := findNodeRuntimeIndexerCLI()
	if err != nil {
		return nil, err
	}
	cfg, err := ftconfig.LoadFromDir(boundaryRoot)
	if err != nil {
		return nil, fmt.Errorf("indexer: load ftconfig: %w", err)
	}
	spawnCmd, err := nodert.BuildBootstrapSpawnCommand(nodert.BootstrapSpawnInput{
		BoundaryRoot:  boundaryRoot,
		Executable:    "node",
		BootstrapPath: cliPath,
		WorkDir:       boundaryRoot,
		Loader:        cfg.Node.Loader,
		ExtraArgs: []string{
			"--root", boundaryRoot,
			"--format", indexCLIFormat,
			"--files", strings.Join(moduleIDs, ","),
		},
	})
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(spawnCmd.Executable, spawnCmd.Args...)
	cmd.Dir = boundaryRoot
	cmd.Env = spawnCmd.Env
	return cmd, nil
}

func findNodeRuntimeIndexerCLI() (string, error) {
	if path, ok := lookPathIndexer("forst-node-index"); ok {
		return path, nil
	}
	candidates := []string{
		filepath.Join(repoRoot(), "packages", "node-runtime", "dist", "indexer", "cli.js"),
		filepath.Join(repoRoot(), "node_modules", "@forst", "node-runtime", "dist", "indexer", "cli.js"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("indexer: node-runtime CLI not found (build packages/node-runtime or install @forst/node-runtime)")
}

func lookPathIndexer(name string) (string, bool) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	if st, err := os.Stat(path); err != nil || st.IsDir() {
		return "", false
	}
	return path, true
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// BoundaryRootFromEntry returns the ftconfig project root for entryDir.
func BoundaryRootFromEntry(entryDir string) (string, error) {
	return boundaryRootFromEntry(entryDir)
}
