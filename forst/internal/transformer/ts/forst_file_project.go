package transformerts

import (
	"fmt"
	"path/filepath"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
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

	tc := typechecker.New(log, false)
	tc.GoWorkspaceDir = goload.FindModuleRoot(paths[0])
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
