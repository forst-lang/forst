package nodeinterop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/ftconfig"
)

var (
	// ErrNotRelativeImport is returned when ResolveTSImport is called with a bare or absolute specifier.
	ErrNotRelativeImport = errors.New("TypeScript import path must be relative (./ or ../)")
	// ErrNoBoundary is returned when no ftconfig.json is found above entryDir.
	ErrNoBoundary = errors.New("TypeScript module outside ftconfig boundary")
	// ErrOutsideBoundary is returned when the resolved file escapes the project boundary.
	ErrOutsideBoundary = errors.New("TypeScript module outside ftconfig boundary")
	// ErrTSModuleNotFound is returned when no .ts/.tsx target exists for the import path.
	ErrTSModuleNotFound = errors.New("TypeScript module not found")
	// ErrResolvedToNonTS is returned when a higher-precedence Forst/Go file shadows the TS target.
	ErrResolvedToNonTS = errors.New("import resolved to Forst/Go module, not TypeScript")
)

// ResolveTSImport resolves a relative TypeScript import path from entryDir.
// Opt-in (// forst:node / import node) must be enforced by the caller.
// moduleId is a POSIX path relative to the discovered ftconfig boundary root.
func ResolveTSImport(entryDir, importPath string) (moduleID string, absPath string, err error) {
	entryDir = filepath.Clean(entryDir)
	path := normalizeImportPath(importPath)
	if path == "" {
		return "", "", fmt.Errorf("%w: empty import path", ErrNotRelativeImport)
	}
	if !isRelativeImport(path) {
		return "", "", ErrNotRelativeImport
	}

	boundaryRoot, err := boundaryRootFromEntry(entryDir)
	if err != nil {
		return "", "", err
	}

	base := filepath.Clean(filepath.Join(entryDir, path))
	candidates := tsResolutionCandidates(base)

	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return "", "", statErr
		}
		if info.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(candidate))
		switch ext {
		case ".ft", ".go":
			return "", "", fmt.Errorf("%w: %s", ErrResolvedToNonTS, filepath.ToSlash(mustRelModuleID(boundaryRoot, candidate)))
		case ".ts", ".tsx":
			absPath = candidate
			resolvedID, err := relModuleID(boundaryRoot, candidate)
			if err != nil {
				return "", "", err
			}
			return resolvedID, absPath, nil
		case ".js":
			continue
		}
	}

	return "", "", fmt.Errorf("%w: %s", ErrTSModuleNotFound, path)
}

// TSImportHintFile returns the resolved .ts/.tsx basename when importPath would select
// TypeScript under entryDir (e.g. "payment.ts"). Empty when no TS target exists or
// a .ft/.go file shadows the import.
func TSImportHintFile(entryDir, importPath string) string {
	_, absPath, err := ResolveTSImport(entryDir, importPath)
	if err != nil {
		return ""
	}
	return filepath.Base(absPath)
}

func boundaryRootFromEntry(entryDir string) (string, error) {
	root, err := ftconfig.BoundaryRootFromDir(entryDir)
	if err != nil {
		return "", ErrNoBoundary
	}
	return root, nil
}

func normalizeImportPath(importPath string) string {
	path := strings.TrimSpace(importPath)
	path = strings.Trim(path, `"'`)
	return path
}

func isRelativeImport(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

// IsBareNodeSpecifier reports whether path is a Node-resolved bare import (e.g. "effect").
func IsBareNodeSpecifier(path string) bool {
	path = normalizeImportPath(path)
	return path != "" && !isRelativeImport(path) && !filepath.IsAbs(path)
}

// tsResolutionCandidates returns candidate paths in opt-in resolution order, stopping before .js files.
func tsResolutionCandidates(base string) []string {
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".ts", ".tsx", ".ft", ".go", ".js":
		return []string{base}
	}

	exts := []string{".ft", ".go", ".ts", ".tsx"}
	var out []string
	for _, tryExt := range exts {
		out = append(out, base+tryExt)
	}
	for _, tryExt := range []string{".ts", ".tsx"} {
		out = append(out, filepath.Join(base, "index"+tryExt))
	}
	return out
}

func relModuleID(boundaryRoot, absPath string) (string, error) {
	boundaryRoot = filepath.Clean(boundaryRoot)
	absPath = filepath.Clean(absPath)

	rel, err := filepath.Rel(boundaryRoot, absPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrOutsideBoundary
	}
	return filepath.ToSlash(rel), nil
}

func mustRelModuleID(boundaryRoot, absPath string) string {
	id, err := relModuleID(boundaryRoot, absPath)
	if err != nil {
		return absPath
	}
	return id
}
