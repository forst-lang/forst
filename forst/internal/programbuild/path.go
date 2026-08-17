package programbuild

import (
	"fmt"
	"strings"
)

// ValidateOutputPath checks that -o is a directory path suitable for forst build.
func ValidateOutputPath(outputPath string) error {
	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf("forst build requires -o <dir> (output directory for bin/<name> and manifest.json)")
	}
	if strings.HasSuffix(strings.ToLower(outputPath), ".go") {
		return fmt.Errorf("forst build -o %q writes a native binary directory, not Go sources; use `forst generate --go-out=%s` (and --go-entry) instead", outputPath, outputPath)
	}
	return nil
}
