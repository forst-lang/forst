package forstdep

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"forst/internal/codegen/layout"
	"forst/internal/goload"
	"forst/internal/gowork"
)

// OverlayDirName returns a filesystem-safe directory name for a module overlay.
// Uses the same injective encoding as layout.OverlayModuleDirName.
func OverlayDirName(modulePath, version string) string {
	return layout.OverlayModuleDirName(modulePath, version)
}

// CopyModuleForOverlay copies a Go module tree into dstOverlayDir for emit.
// Skips .ft, *.gen.go, *_test.go, and common junk dirs.
func CopyModuleForOverlay(srcModuleDir, dstOverlayDir string) error {
	srcModuleDir = filepath.Clean(srcModuleDir)
	dstOverlayDir = filepath.Clean(dstOverlayDir)
	if srcModuleDir == "" || dstOverlayDir == "" {
		return fmt.Errorf("overlay copy: empty src or dst")
	}
	if err := os.MkdirAll(dstOverlayDir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(srcModuleDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcModuleDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" || name == ".forst" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dstOverlayDir, rel), 0o755)
		}
		name := d.Name()
		if strings.HasSuffix(name, ".ft") {
			return nil
		}
		if strings.HasSuffix(name, ".gen.go") || strings.HasPrefix(name, "z_forst_") {
			return nil
		}
		if strings.HasSuffix(name, "_test.go") {
			return nil
		}
		return copyFile(path, filepath.Join(dstOverlayDir, rel))
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// OverlayModule describes a copied+emitted dependency module under .forst/overlay.
type OverlayModule struct {
	ModulePath string
	Dir        string // absolute overlay root
}

// BuildOverlayRoots copies each unique ModuleDir into boundary/.forst/overlay/...
func BuildOverlayRoots(boundary string, pkgs []DiscoveredPackage) ([]OverlayModule, error) {
	if boundary == "" || len(pkgs) == 0 {
		return nil, nil
	}
	root := layout.NewRoot(boundary)
	byModule := make(map[string]goload.PackageLoc)
	for i := range pkgs {
		p := &pkgs[i]
		mp := p.Loc.ModulePath
		md := p.Loc.ModuleDir
		if mp == "" || md == "" {
			md = goload.FindModuleRoot(p.Loc.Dir)
			mp = goload.ModulePath(md)
			if mp == "" || md == "" {
				continue
			}
			p.Loc.ModulePath = mp
			p.Loc.ModuleDir = md
		}
		if _, ok := byModule[mp]; ok {
			continue
		}
		byModule[mp] = p.Loc
	}
	var out []OverlayModule
	for mp, loc := range byModule {
		dst := root.OverlayModule(mp, loc.ModuleVersion)
		if err := os.RemoveAll(dst); err != nil {
			return nil, err
		}
		if err := CopyModuleForOverlay(loc.ModuleDir, dst); err != nil {
			return nil, err
		}
		out = append(out, OverlayModule{ModulePath: mp, Dir: dst})
	}
	return out, nil
}

// OverlayReplaces builds go.mod replace entries for overlay modules.
func OverlayReplaces(overlays []OverlayModule) []gowork.PackageReplace {
	out := make([]gowork.PackageReplace, 0, len(overlays))
	for _, o := range overlays {
		if o.ModulePath == "" || o.Dir == "" {
			continue
		}
		out = append(out, gowork.PackageReplace{ImportPath: o.ModulePath, Dir: o.Dir})
	}
	return out
}

// OverlayPkgDir returns the overlay directory for an import path's package dir.
func OverlayPkgDir(overlayRoot, moduleDir, pkgDir string) (string, error) {
	rel, err := filepath.Rel(moduleDir, pkgDir)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return overlayRoot, nil
	}
	return filepath.Join(overlayRoot, rel), nil
}
