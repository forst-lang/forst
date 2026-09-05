// Package unixpath shortens AF_UNIX socket paths that exceed platform limits.
package unixpath

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"runtime"
)

// MaxLen stays under the macOS AF_UNIX path limit (104 bytes).
const MaxLen = 100

// EnsureLength returns abs unchanged when short enough, otherwise a stable
// /tmp/<tmpPrefix><hash8>.sock path. tmpPrefix should end with "-" (e.g. "forst-inv-").
func EnsureLength(abs, tmpPrefix string) string {
	if runtime.GOOS == "windows" || abs == "" {
		return abs
	}
	if len(abs) <= MaxLen {
		return abs
	}
	if tmpPrefix == "" {
		tmpPrefix = "forst-"
	}
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join("/tmp", fmt.Sprintf("%s%x.sock", tmpPrefix, sum[:8]))
}
