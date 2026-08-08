package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// generateWriteStats accumulates how many emit paths were rewritten vs left alone.
type generateWriteStats struct {
	Written int
	Skipped int
}

// writeGeneratedFile writes content atomically when bytes differ from the file on disk.
// When stats is non-nil it increments Written or Skipped.
func writeGeneratedFile(path string, content []byte, stats *generateWriteStats) error {
	written, err := writeFileAtomic(path, content)
	if err != nil {
		return err
	}
	if stats != nil {
		if written {
			stats.Written++
		} else {
			stats.Skipped++
		}
	}
	return nil
}

// writeFileAtomic writes data to path via a same-directory temp file and rename.
// It returns written=false when the file on disk already has identical bytes.
func writeFileAtomic(path string, data []byte) (written bool, err error) {
	existing, readErr := generateIO.ReadFile(path)
	if readErr == nil && bytes.Equal(existing, data) {
		return false, nil
	}

	dir := filepath.Dir(path)
	if err := generateIO.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create directory for %s: %w", path, err)
	}

	tmpPath, err := generateTempPath(path)
	if err != nil {
		return false, err
	}

	if err := generateIO.WriteFile(tmpPath, data, 0644); err != nil {
		return false, fmt.Errorf("failed to write temp file for %s: %w", path, err)
	}

	syncFileBestEffort(tmpPath)

	if err := generateIO.Rename(tmpPath, path); err != nil {
		_ = generateIO.Remove(tmpPath)
		return false, fmt.Errorf("failed to rename temp file over %s: %w", path, err)
	}
	return true, nil
}

// generateTempPath returns a unique sibling path for an atomic write.
func generateTempPath(target string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate temp name for %s: %w", target, err)
	}
	return target + ".tmp-" + hex.EncodeToString(b[:]), nil
}

// syncFileBestEffort fsyncs path when the real filesystem is available.
func syncFileBestEffort(path string) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return
	}
	_ = f.Sync()
	_ = f.Close()
}
