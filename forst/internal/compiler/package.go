package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/forstpkg"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// goWorkspaceDirForCheck returns the directory used for go/packages when checking Forst↔Go imports.
func (c *Compiler) goWorkspaceDirForCheck() string {
	if c.Args.PackageRoot != "" {
		return goload.GoWorkspaceForPackages(c.Args.PackageRoot)
	}
	return goload.GoWorkspaceForPackages(c.Args.FilePath)
}

// collectSamePackageFtPaths finds all .ft files under rootDir that declare the same package
// as entryPath (parsed). Paths are sorted for stable merges.
func collectSamePackageFtPaths(log *logrus.Logger, rootDir, entryPath string) ([]string, error) {
	entryNodes, err := forstpkg.ParseForstFile(log, entryPath)
	if err != nil {
		return nil, fmt.Errorf("parse entry file: %w", err)
	}
	pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(entryNodes))

	rootDir = filepath.Clean(rootDir)
	var out []string
	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".ft") {
			return nil
		}
		nodes, err := forstpkg.ParseForstFile(log, path)
		if err != nil {
			return nil
		}
		if forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes)) == pkg {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no .ft files for package %q under %s", pkg, rootDir)
	}
	sort.Strings(out)
	return out, nil
}

func entryContainedInRoot(root, entry string) error {
	root = filepath.Clean(root)
	entry = filepath.Clean(entry)
	rel, err := filepath.Rel(root, entry)
	if err != nil {
		return fmt.Errorf("entry file must be under -root: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("entry file %q is not under -root %q", entry, root)
	}
	return nil
}

// loadMergedPackageAST parses and merges all same-package .ft files under PackageRoot.
func (c *Compiler) loadMergedPackageAST() ([]ast.Node, error) {
	root, err := filepath.Abs(c.Args.PackageRoot)
	if err != nil {
		return nil, err
	}
	entry, err := filepath.Abs(c.Args.FilePath)
	if err != nil {
		return nil, err
	}
	if err := entryContainedInRoot(root, entry); err != nil {
		return nil, err
	}
	paths, err := collectSamePackageFtPaths(c.log, root, entry)
	if err != nil {
		return nil, err
	}
	merged, _, err := forstpkg.ParseAndMergePackage(c.log, paths)
	if err != nil {
		return nil, err
	}
	c.log.Debugf("Merged %d Forst file(s) for package compile", len(paths))
	return merged, nil
}

// lexParseEntryFile lexes and parses the single input file (traditional compile path).
func (c *Compiler) lexParseEntryFile() ([]ast.Node, error) {
	source, err := c.readSourceFile()
	if err != nil {
		return nil, err
	}

	c.reportPhase("Performing lexical analysis...")
	memBefore := getMemStats()

	l := lexer.New(source, c.Args.FilePath, c.log)
	tokens := l.Lex()

	memAfter := getMemStats()
	c.logMemUsage("lexical analysis", memBefore, memAfter)

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintTokens(tokens)
	}

	c.reportPhase("Performing syntax analysis...")
	memBefore = getMemStats()

	psr := parser.New(tokens, c.Args.FilePath, c.log)
	forstNodes, err := psr.ParseFile()
	if err != nil {
		return nil, err
	}

	memAfter = getMemStats()
	c.logMemUsage("syntax analysis", memBefore, memAfter)

	if c.Args.LogLevel == "debug" || c.Args.LogLevel == "trace" {
		c.debugPrintForstAST(forstNodes)
	}

	return forstNodes, nil
}
