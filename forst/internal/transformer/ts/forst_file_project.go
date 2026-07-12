package transformerts

import (
	"fmt"
	"path/filepath"
	"sort"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/modulecheck"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// ForstFileChunk holds one parsed .ft file for merged-package generation.
type ForstFileChunk struct {
	Path  string
	Stem  string
	Nodes []ast.Node
}

// ParseMergedTypecheckProject parses the full module set (same paths `forst generate` / ftconfig
// discovery already collected), merges ASTs with forstpkg.MergePackageASTs like `forst run -root`,
// and runs a single typecheck pass so cross-file types and functions resolve.
// Path order is preserved; semantic analysis does not depend on file merge order.
func ParseMergedTypecheckProject(filePaths []string, log *logrus.Logger) ([]ForstFileChunk, *typechecker.TypeChecker, error) {
	if len(filePaths) == 0 {
		return nil, nil, fmt.Errorf("no Forst files to parse")
	}
	paths := append([]string(nil), filePaths...)

	merged, byPath, err := forstpkg.ParseAndMergePackage(log, paths)
	if err != nil {
		return nil, nil, err
	}

	moduleRoot := goload.FindModuleRoot(paths[0])
	entryDir := filepath.Dir(paths[0])

	tc := typechecker.New(log, false)
	tc.ConfigureForForstFile(moduleRoot, entryDir, merged)
	if err := tc.CheckTypes(merged); err != nil {
		return nil, nil, fmt.Errorf("failed to type check: %w", err)
	}

	chunks := make([]ForstFileChunk, 0, len(paths))
	for _, p := range paths {
		nodes := byPath[p]
		base := filepath.Base(p)
		stem := base[:len(base)-len(filepath.Ext(base))]
		chunks = append(chunks, ForstFileChunk{Path: p, Stem: stem, Nodes: nodes})
	}
	return chunks, tc, nil
}

// GenerateTSOptions configures per-run TypeScript emission (forst generate).
type GenerateTSOptions struct {
	GenerateStreamingClients bool
}

// ForstFileStem returns the basename of path without the .ft extension.
func ForstFileStem(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext == "" {
		return base
	}
	return base[:len(base)-len(ext)]
}

// StemMatchesPackage reports whether a .ft file path aligns with its declared package name.
// The file stem may equal the package, or the parent directory name may equal the package
// (e.g. auth/login.ft with package auth).
func StemMatchesPackage(filePath, pkg string) bool {
	if ForstFileStem(filePath) == pkg {
		return true
	}
	parent := filepath.Base(filepath.Dir(filePath))
	return parent == pkg
}

// ValidateDiscoveredFileStems returns an error when any discovered file stem and parent
// directory disagree with its declared package (forst generate default).
func ValidateDiscoveredFileStems(filePaths []string, allowMismatch bool, log *logrus.Logger) error {
	if allowMismatch {
		return nil
	}
	for _, path := range filePaths {
		nodes, err := forstpkg.ParseForstFile(log, path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
		if StemMatchesPackage(path, pkg) {
			continue
		}
		return fmt.Errorf("%s: file stem %q must match declared package %q (rename the file, move it under a %q/ directory, or pass --allow-stem-package-mismatch)",
			path, ForstFileStem(path), pkg, pkg)
	}
	return nil
}

// GenerateTypeScriptOutputsByPackage typechecks discovered files per Forst package via
// modulecheck and emits one TypeScriptOutput per package (merged AST, package-keyed client stem).
func GenerateTypeScriptOutputsByPackage(filePaths []string, log *logrus.Logger, opts *GenerateTSOptions) ([]*TypeScriptOutput, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no Forst files to parse")
	}
	if opts == nil {
		opts = &GenerateTSOptions{}
	}

	paths := append([]string(nil), filePaths...)
	sort.Strings(paths)

	parsed := make(map[string][]ast.Node, len(paths))
	for _, p := range paths {
		nodes, err := forstpkg.ParseForstFile(log, p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		parsed[p] = nodes
	}

	moduleRoot := goload.FindModuleRoot(paths[0])
	if moduleRoot == "" {
		moduleRoot = filepath.Dir(paths[0])
	}

	scan, err := modulecheck.ScanModule(log, modulecheck.Options{
		ModuleRoot:   moduleRoot,
		ParsedFiles:  parsed,
		SkipGoLoad:   true,
		SkipValidate: true,
	})
	if err != nil {
		return nil, err
	}
	modResult := scan.Result()

	packageNames := packagesWithDiscoveredFiles(modResult.ForstPkgToFiles, paths)
	outputs := make([]*TypeScriptOutput, 0, len(packageNames))
	for _, pkgName := range packageNames {
		pkgPaths := packageDiscoveredPaths(modResult.ForstPkgToFiles[pkgName], paths)
		nodes := modResult.PerPackageNodes[pkgName]
		if nodes == nil {
			return nil, fmt.Errorf("package %q missing merged AST", pkgName)
		}

		tc := typechecker.New(log, false)
		tc.ConfigureForForstFile(moduleRoot, filepath.Dir(pkgPaths[0]), nodes)
		tc.SetModuleResult(modResult)
		if err := tc.CheckTypes(nodes); err != nil {
			return nil, fmt.Errorf("package %s: %w", pkgName, err)
		}

		tr := New(tc, log)
		tr.GenerateStreamingClients = opts.GenerateStreamingClients
		out, err := tr.TransformForstFileToTypeScript(nodes, pkgName)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", pkgName, err)
		}
		outputs = append(outputs, out)
	}
	return outputs, nil
}

func packageDiscoveredPaths(pkgPaths, discovered []string) []string {
	inScope := make(map[string]struct{}, len(discovered))
	for _, p := range discovered {
		inScope[filepath.Clean(p)] = struct{}{}
	}
	var out []string
	for _, p := range pkgPaths {
		if _, ok := inScope[filepath.Clean(p)]; ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func packagesWithDiscoveredFiles(byPackage map[string][]string, discovered []string) []string {
	inScope := make(map[string]struct{}, len(discovered))
	for _, p := range discovered {
		inScope[filepath.Clean(p)] = struct{}{}
	}
	var names []string
	for pkg, paths := range byPackage {
		for _, p := range paths {
			if _, ok := inScope[filepath.Clean(p)]; ok {
				names = append(names, pkg)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}

// PackageHasRunnableExports reports whether out has public invoke-eligible functions.
func PackageHasRunnableExports(out *TypeScriptOutput) bool {
	return out != nil && len(out.Functions) > 0
}

// GenerateTypeScriptOutputsPerFile runs the TypeScript transformer per file using a shared
// typechecker (from ParseMergedTypecheckProject).
func GenerateTypeScriptOutputsPerFile(chunks []ForstFileChunk, tc *typechecker.TypeChecker, log *logrus.Logger, opts *GenerateTSOptions) ([]*TypeScriptOutput, error) {
	if opts == nil {
		opts = &GenerateTSOptions{}
	}
	outputs := make([]*TypeScriptOutput, 0, len(chunks))
	for _, ch := range chunks {
		tr := New(tc, log)
		tr.GenerateStreamingClients = opts.GenerateStreamingClients
		out, err := tr.TransformForstFileToTypeScript(ch.Nodes, ch.Stem)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", ch.Path, err)
		}
		outputs = append(outputs, out)
	}
	return outputs, nil
}
