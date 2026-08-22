package nodeinterop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"forst/internal/ftconfig"
)

// PrecompileEntries bundles each source module entry to host-neutral ESM under outDir.
func PrecompileEntries(boundaryRoot string, sourceModuleIDs []string, outDir string) error {
	boundaryRoot = filepath.Clean(boundaryRoot)
	if boundaryRoot == "" {
		return fmt.Errorf("precompile: boundaryRoot is required")
	}
	outDir = strings.TrimSpace(outDir)
	if outDir == "" {
		outDir = ".forst/js"
	}
	esbuild, err := findEsbuild(boundaryRoot)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, src := range sourceModuleIDs {
		src = filepath.ToSlash(strings.TrimSpace(src))
		if src == "" {
			continue
		}
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		entry := filepath.Join(boundaryRoot, filepath.FromSlash(src))
		if st, statErr := os.Stat(entry); statErr != nil || st.IsDir() {
			return fmt.Errorf("precompile: source module not found: %s", src)
		}
		outRel := ftconfig.RuntimeModuleID(src, outDir, ftconfig.LegacyModuleCompiled)
		outFile := filepath.Join(boundaryRoot, filepath.FromSlash(outRel))
		if err := os.MkdirAll(filepath.Dir(outFile), 0o750); err != nil {
			return fmt.Errorf("precompile: mkdir %s: %w", filepath.Dir(outFile), err)
		}
		cmd := exec.Command(esbuild, entry,
			"--bundle",
			"--format=esm",
			"--outfile="+outFile,
		)
		cmd.Dir = boundaryRoot
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = runErr.Error()
			}
			return fmt.Errorf("precompile %s: %s", src, msg)
		}
	}
	return nil
}

func findEsbuild(boundaryRoot string) (string, error) {
	candidates := []string{
		filepath.Join(boundaryRoot, "node_modules", ".bin", "esbuild"),
	}
	if path, err := exec.LookPath("esbuild"); err == nil {
		candidates = append([]string{path}, candidates...)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("precompile: esbuild not found (install esbuild in the project or on PATH)")
}

// RuntimeModuleID maps source moduleId to runtime path for precompiled artifact mode.
func RuntimeModuleID(sourceID, outDir string) string {
	return ftconfig.RuntimeModuleID(sourceID, outDir, ftconfig.LegacyModuleCompiled)
}

// RemapManifestModuleIDs returns a copy of manifest with export moduleIds rewritten for runtime.
func RemapManifestModuleIDs(m ManifestV1, outDir string) ManifestV1 {
	out := m
	out.Exports = make([]ExportEntry, len(m.Exports))
	for i, exp := range m.Exports {
		out.Exports[i] = exp
		out.Exports[i].ModuleID = ftconfig.RuntimeModuleID(exp.ModuleID, outDir, ftconfig.LegacyModuleCompiled)
	}
	return out
}

// CopyJSArtifacts copies precompiled output tree into destDir preserving relative layout.
func CopyJSArtifacts(boundaryRoot, destDir, outDirRel string) error {
	src := filepath.Join(boundaryRoot, filepath.FromSlash(outDirRel))
	dst := filepath.Join(destDir, filepath.FromSlash(outDirRel))
	return copyDir(src, dst)
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("copy js artifacts: %s is not a directory", src)
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
