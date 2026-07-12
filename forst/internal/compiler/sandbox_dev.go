package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"forst/internal/codegen/layout"
)

const devSandboxDirName = "dev"

// DevSandboxDir returns the stable dev reload sandbox under boundaryRoot.
func DevSandboxDir(boundaryRoot string) string {
	layoutRoot := layout.NewRoot(boundaryRoot)
	return filepath.Join(layoutRoot.Boundary, ".forst", "run", devSandboxDirName)
}

// DevBinPath returns the prebuilt binary path for dev reload.
func DevBinPath(boundaryRoot string) string {
	return filepath.Join(DevSandboxDir(boundaryRoot), "bin")
}

// IsDevBinPath reports whether path is the stable dev reload binary.
func IsDevBinPath(path, boundaryRoot string) bool {
	if path == "" || boundaryRoot == "" {
		return false
	}
	absBin, err := filepath.Abs(DevBinPath(boundaryRoot))
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return absPath == absBin
}

// SandboxModCache skips redundant go mod tidy when go.mod content is unchanged.
type SandboxModCache struct {
	mu       sync.Mutex
	lastHash string
}

// NeedsTidy reports whether go mod tidy should run for goModPath.
func (c *SandboxModCache) NeedsTidy(goModPath string) (bool, error) {
	if c == nil {
		return true, nil
	}
	hash, err := hashFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return true, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return hash != c.lastHash, nil
}

// Record stores the current go.mod hash after a successful tidy.
func (c *SandboxModCache) Record(goModPath string) error {
	if c == nil {
		return nil
	}
	hash, err := hashFile(goModPath)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.lastHash = hash
	c.mu.Unlock()
	return nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

type sandboxWriteOpts struct {
	stableDir     bool
	modTidyCache  *SandboxModCache
	sandboxTiming *CompileSandboxTiming
}

func resolveSandboxDir(boundaryRoot string, stable bool) (string, error) {
	if boundaryRoot == "" {
		return "", fmt.Errorf("boundary root required for sandbox")
	}
	layoutRoot := layout.NewRoot(boundaryRoot)
	runBase := filepath.Join(layoutRoot.Boundary, ".forst", "run")
	if err := os.MkdirAll(runBase, 0o755); err != nil {
		return "", err
	}
	if stable {
		tempDir := DevSandboxDir(boundaryRoot)
		if err := os.MkdirAll(tempDir, 0o755); err != nil {
			return "", err
		}
		return tempDir, nil
	}
	return mkdirTemp(runBase, "forst-*")
}
