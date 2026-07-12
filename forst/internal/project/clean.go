package project

import (
	"os"
	"path/filepath"
)

// GeneratedDirName is the boundary-relative directory for compiler-generated artifacts.
const GeneratedDirName = ".forst"

// CleanResult describes paths removed (or that would be removed) by forst clean.
type CleanResult struct {
	BoundaryRoot string
	Removed      []string
	DryRun       bool
}

// CleanGenerated removes compiler-generated output under boundaryRoot/.forst.
// This includes run sandboxes, test emit dirs, executor temp modules, nodert sockets,
// and invoke ready markers. User config such as .forst-gomod/ is not touched.
func CleanGenerated(boundaryRoot string, dryRun bool) (CleanResult, error) {
	abs, err := filepath.Abs(boundaryRoot)
	if err != nil {
		return CleanResult{}, err
	}
	target := filepath.Join(abs, GeneratedDirName)
	result := CleanResult{
		BoundaryRoot: abs,
		DryRun:       dryRun,
	}
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}
	if dryRun {
		result.Removed = []string{target}
		return result, nil
	}
	if err := os.RemoveAll(target); err != nil {
		return result, err
	}
	result.Removed = []string{target}
	return result, nil
}
