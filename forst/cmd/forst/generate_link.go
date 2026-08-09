package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

const forstGeneratedMarkerName = ".forst-generated"

// forstGeneratedMarker records which ftconfig boundary owns a generated outDir.
type forstGeneratedMarker struct {
	BoundaryRoot string `json:"boundaryRoot"`
	PackageName  string `json:"packageName"`
}

// generateLinkIO hooks filesystem operations for node_modules linking tests.
var generateLinkIO = struct {
	Symlink      func(target, link string) error
	Junction     func(target, link string) error
	Lstat        func(name string) (os.FileInfo, error)
	Stat         func(name string) (os.FileInfo, error)
	RemoveAll    func(path string) error
	Readlink     func(name string) (string, error)
	MkdirAll     func(path string, perm os.FileMode) error
	WriteFile    func(name string, data []byte, perm os.FileMode) error
	ReadFile     func(name string) ([]byte, error)
	CopyDir      func(src, dst string) error
	EvalSymlinks func(path string) (string, error)
	GOOS         string
}{
	Symlink:      os.Symlink,
	Junction:     createJunction,
	Lstat:        os.Lstat,
	Stat:         os.Stat,
	RemoveAll:    os.RemoveAll,
	Readlink:     os.Readlink,
	MkdirAll:     os.MkdirAll,
	WriteFile:    os.WriteFile,
	ReadFile:     os.ReadFile,
	CopyDir:      copyDirRecursive,
	EvalSymlinks: filepath.EvalSymlinks,
	GOOS:         runtime.GOOS,
}

// maybeLinkGeneratedClient links when genCfg.ShouldLink is true.
func maybeLinkGeneratedClient(boundaryRoot, outDir string, genCfg ftconfig.GenerateConfig, log *logrus.Logger) error {
	if !genCfg.ShouldLink(boundaryRoot) {
		log.WithFields(logrus.Fields{
			"packageName": genCfg.PackageName,
			"outDir":      outDir,
			"link":        genCfg.Link,
			"reason":      "generate.link disables linking",
		}).Debug("skipping node_modules link")
		return nil
	}
	return linkGeneratedClient(boundaryRoot, outDir, genCfg.PackageName, log)
}

// linkGeneratedClient creates or refreshes node_modules/<packageName> -> outDir.
// Caller is expected to gate on ShouldLink. This function assumes linking is desired.
func linkGeneratedClient(boundaryRoot, outDir, packageName string, log *logrus.Logger) error {
	boundaryRoot = filepath.Clean(boundaryRoot)
	outDir = filepath.Clean(outDir)
	if abs, err := filepath.Abs(boundaryRoot); err == nil {
		boundaryRoot = abs
	}
	if abs, err := filepath.Abs(outDir); err == nil {
		outDir = abs
	}

	nodeModules, ok := findNodeModulesDir(boundaryRoot)
	if !ok {
		log.WithFields(logrus.Fields{
			"boundaryRoot": boundaryRoot,
			"packageName":  packageName,
		}).Warn("skipping node_modules link: no node_modules directory found above boundary root")
		return nil
	}

	pnpPath := filepath.Join(filepath.Dir(nodeModules), ".pnp.cjs")
	if _, err := generateLinkIO.Stat(pnpPath); err == nil {
		log.WithFields(logrus.Fields{
			"boundaryRoot": boundaryRoot,
			"packageName":  packageName,
			"pnp":          pnpPath,
		}).Warn("skipping node_modules link: Yarn Plug'n'Play detected (.pnp.cjs). Use committed mode (outDir outside .forst, link \"never\") with a file: or workspace dependency instead")
		return nil
	}

	pnpmStore := filepath.Join(nodeModules, ".pnpm")
	if _, err := generateLinkIO.Stat(pnpmStore); err == nil {
		log.WithFields(logrus.Fields{
			"boundaryRoot": boundaryRoot,
			"packageName":  packageName,
			"pnpmStore":    pnpmStore,
		}).Warn("pnpm isolated node_modules detected (.pnpm store). Ephemeral linking may not resolve reliably; use committed mode (outDir outside .forst, link \"never\") or ensure postinstall runs forst generate")
	}

	linkPath := filepath.Join(nodeModules, filepath.FromSlash(packageName))
	if err := generateLinkIO.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return fmt.Errorf("create scope directory for %s: %w", packageName, err)
	}

	action, err := ensureGeneratedClientLink(linkPath, outDir, boundaryRoot, packageName, log)
	if err != nil {
		return err
	}

	log.WithFields(logrus.Fields{
		"packageName": packageName,
		"linkPath":    linkPath,
		"target":      outDir,
		"action":      action,
	}).Debug("node_modules link ready")
	return nil
}

// ensureGeneratedClientLink inspects linkPath and creates, replaces, or leaves it.
// Returns action unchanged|created|replaced.
func ensureGeneratedClientLink(linkPath, outDir, boundaryRoot, packageName string, log *logrus.Logger) (string, error) {
	info, err := generateLinkIO.Lstat(linkPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat link path %s: %w", linkPath, err)
		}
		// Marker first so a copy fallback includes ownership metadata.
		if err := writeForstGeneratedMarker(outDir, boundaryRoot, packageName, nil); err != nil {
			return "", err
		}
		if err := createDirLink(outDir, linkPath, log); err != nil {
			return "", err
		}
		return "created", nil
	}

	if isSymlinkOrJunction(info, linkPath) {
		resolved, resErr := resolveLinkTarget(linkPath)
		if resErr != nil {
			return "", fmt.Errorf("resolve existing link %s: %w", linkPath, resErr)
		}
		if sameResolvedPath(resolved, outDir) {
			return "unchanged", nil
		}
		if err := assertLinkReplaceable(resolved, linkPath, boundaryRoot, packageName, true); err != nil {
			return "", err
		}
		if err := generateLinkIO.RemoveAll(linkPath); err != nil {
			return "", fmt.Errorf("remove existing link %s: %w", linkPath, err)
		}
		if err := writeForstGeneratedMarker(outDir, boundaryRoot, packageName, nil); err != nil {
			return "", err
		}
		if err := createDirLink(outDir, linkPath, log); err != nil {
			return "", err
		}
		return "replaced", nil
	}

	if !info.IsDir() {
		return "", fmt.Errorf("generate: %s exists and is not a Forst-managed link or directory; set generate.packageName to a different name", linkPath)
	}

	// Real directory (copy fallback or foreign). Ownership is the marker inside it.
	if err := assertLinkReplaceable(linkPath, linkPath, boundaryRoot, packageName, false); err != nil {
		return "", err
	}
	if err := generateLinkIO.RemoveAll(linkPath); err != nil {
		return "", fmt.Errorf("remove existing directory %s: %w", linkPath, err)
	}
	if err := writeForstGeneratedMarker(outDir, boundaryRoot, packageName, nil); err != nil {
		return "", err
	}
	if err := createDirLink(outDir, linkPath, log); err != nil {
		return "", err
	}
	return "replaced", nil
}

// assertLinkReplaceable checks .forst-generated ownership at markerDir.
// isSymlinkTarget is true when markerDir is a symlink target; false when markerDir is the link path itself.
func assertLinkReplaceable(markerDir, linkPath, boundaryRoot, packageName string, isSymlinkTarget bool) error {
	marker, err := readForstGeneratedMarker(markerDir)
	if err != nil {
		if isSymlinkTarget {
			return fmt.Errorf("generate: %s exists and is not a Forst-managed link; set generate.packageName to a different name", linkPath)
		}
		return fmt.Errorf("generate: %s is a real directory that Forst did not create; set generate.packageName to a different name", linkPath)
	}
	existing := filepath.Clean(marker.BoundaryRoot)
	current := filepath.Clean(boundaryRoot)
	if !sameResolvedPath(existing, current) {
		return fmt.Errorf("generate: node_modules/%s already belongs to another Forst project\n  existing: %s\n  current:  %s\n  set generate.packageName in one of them",
			packageName, existing, current)
	}
	return nil
}

func isSymlinkOrJunction(info os.FileInfo, path string) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	// Windows junctions often look like directories; Readlink succeeds for them.
	if _, err := generateLinkIO.Readlink(path); err == nil {
		return true
	}
	return false
}

func resolveLinkTarget(linkPath string) (string, error) {
	target, err := generateLinkIO.Readlink(linkPath)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(linkPath), target)
	}
	return resolvePath(target), nil
}

func resolvePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if eval, err := generateLinkIO.EvalSymlinks(abs); err == nil {
		return eval
	}
	return filepath.Clean(abs)
}

func sameResolvedPath(a, b string) bool {
	return resolvePath(a) == resolvePath(b)
}

func findNodeModulesDir(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		candidate := filepath.Join(dir, "node_modules")
		st, err := generateLinkIO.Stat(candidate)
		if err == nil && st.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func createDirLink(target, linkPath string, log *logrus.Logger) error {
	goos := generateLinkIO.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}

	var linkErr error
	if goos == "windows" {
		if generateLinkIO.Junction != nil {
			linkErr = generateLinkIO.Junction(target, linkPath)
			if linkErr == nil {
				return nil
			}
		} else {
			linkErr = fmt.Errorf("junction not available")
		}
	} else {
		linkErr = generateLinkIO.Symlink(target, linkPath)
		if linkErr == nil {
			return nil
		}
	}

	log.WithFields(logrus.Fields{
		"linkPath": linkPath,
		"target":   target,
		"error":    linkErr.Error(),
	}).Warn("directory link creation failed; falling back to a recursive copy (stale until next generate)")
	if err := generateLinkIO.CopyDir(target, linkPath); err != nil {
		return fmt.Errorf("copy fallback for %s -> %s: %w", linkPath, target, err)
	}
	return nil
}

// writeForstGeneratedMarker writes ownership metadata inside outDir via the atomic writer.
func writeForstGeneratedMarker(outDir, boundaryRoot, packageName string, stats *generateWriteStats) error {
	marker := forstGeneratedMarker{
		BoundaryRoot: filepath.Clean(boundaryRoot),
		PackageName:  packageName,
	}
	if abs, err := filepath.Abs(marker.BoundaryRoot); err == nil {
		marker.BoundaryRoot = abs
	}
	data, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", forstGeneratedMarkerName, err)
	}
	path := filepath.Join(outDir, forstGeneratedMarkerName)
	if err := writeGeneratedFile(path, append(data, '\n'), stats); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// readForstGeneratedMarker reads ownership metadata from outDir.
func readForstGeneratedMarker(outDir string) (forstGeneratedMarker, error) {
	path := filepath.Join(outDir, forstGeneratedMarkerName)
	data, err := generateLinkIO.ReadFile(path)
	if err != nil {
		return forstGeneratedMarker{}, err
	}
	var marker forstGeneratedMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return forstGeneratedMarker{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if strings.TrimSpace(marker.BoundaryRoot) == "" {
		return forstGeneratedMarker{}, fmt.Errorf("%s missing boundaryRoot", path)
	}
	return marker, nil
}

// createJunction is implemented in generate_link_windows.go (Windows) and generate_link_unix.go (!Windows).

func copyDirRecursive(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			linkTarget, readErr := os.Readlink(path)
			if readErr != nil {
				return readErr
			}
			return os.Symlink(linkTarget, target)
		}
		return copyFileContents(path, target)
	})
}

func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
