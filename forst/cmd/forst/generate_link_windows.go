//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// createJunction creates a directory junction on Windows via mklink /J.
// Junctions do not require elevation for local paths (unlike symlinks).
func createJunction(target, link string) error {
	target, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	link, err = filepath.Abs(link)
	if err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J %q %q: %w (%s)", link, target, err, string(out))
	}
	return nil
}

// ensureJunctionTargetAbsent removes a stale link path before mklink when needed.
func ensureJunctionTargetAbsent(link string) error {
	if _, err := os.Lstat(link); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(link)
}
