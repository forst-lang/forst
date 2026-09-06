// Package depoverlay copies external Forst modules into .forst/overlay and emits *.gen.go.
package depoverlay

import (
	"fmt"
	"os"
	"path/filepath"

	"forst/internal/ast"
	"forst/internal/codegen/layout"
	"forst/internal/forstdep"
	"forst/internal/generators"
	"forst/internal/goload"
	"forst/internal/gowork"
	"forst/internal/modulecheck"
	transformer_go "forst/internal/transformer/go"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// Emit copies import-reachable external Forst modules into boundary/.forst/overlay,
// emits *.gen.go for each package, and returns go.mod replace directives.
// Never writes into GOMODCACHE.
func Emit(log *logrus.Logger, boundary string, modResult *modulecheck.ModuleResult, exportStructFields bool) ([]gowork.PackageReplace, error) {
	if modResult == nil || len(modResult.ExternalImports) == 0 || boundary == "" {
		return nil, nil
	}
	pkgs := make([]forstdep.DiscoveredPackage, 0, len(modResult.ExternalImports))
	for importPath := range modResult.ExternalImports {
		loc, ok := modResult.ExternalLocs[importPath]
		if !ok {
			continue
		}
		pkgs = append(pkgs, forstdep.DiscoveredPackage{
			Loc:      loc,
			ForstPkg: modResult.ImportPathToForstPkg()[importPath],
			Nodes:    modResult.ExternalNodes[importPath],
		})
	}
	overlays, err := forstdep.BuildOverlayRoots(boundary, pkgs)
	if err != nil {
		return nil, err
	}
	byModule := make(map[string]string, len(overlays))
	for _, o := range overlays {
		byModule[o.ModulePath] = o.Dir
	}
	for _, p := range pkgs {
		mp := p.Loc.ModulePath
		md := p.Loc.ModuleDir
		if mp == "" || md == "" {
			md = goload.FindModuleRoot(p.Loc.Dir)
			mp = goload.ModulePath(md)
		}
		overlayRoot := byModule[mp]
		if overlayRoot == "" {
			continue
		}
		pkgDir, err := forstdep.OverlayPkgDir(overlayRoot, md, p.Loc.Dir)
		if err != nil {
			return nil, err
		}
		tc := modResult.PerImportPath[p.Loc.ImportPath]
		nodes := modResult.ExternalNodes[p.Loc.ImportPath]
		if tc == nil || len(nodes) == 0 {
			continue
		}
		if err := emitPackage(log, tc, nodes, pkgDir, p.ForstPkg, exportStructFields, modResult); err != nil {
			return nil, fmt.Errorf("emit %s: %w", p.Loc.ImportPath, err)
		}
	}
	return forstdep.OverlayReplaces(overlays), nil
}

func emitPackage(log *logrus.Logger, tc *typechecker.TypeChecker, nodes []ast.Node, overlayPkgDir, pkgName string, exportStructFields bool, modResult *modulecheck.ModuleResult) error {
	if err := os.MkdirAll(overlayPkgDir, 0o755); err != nil {
		return err
	}
	tr := transformer_go.New(tc, log, exportStructFields)
	if modResult != nil {
		tr.SetModuleResult(modResult)
	}
	goAST, err := tr.TransformForstFileToGo(nodes)
	if err != nil {
		return err
	}
	code, err := generators.GenerateGoCode(goAST)
	if err != nil {
		return err
	}
	if pkgName == "" {
		pkgName = tc.ForstPackage()
	}
	if pkgName == "" {
		pkgName = "pkg"
	}
	outPath := filepath.Join(overlayPkgDir, pkgName+layout.SuffixGen)
	return os.WriteFile(outPath, []byte(code), 0o644)
}
