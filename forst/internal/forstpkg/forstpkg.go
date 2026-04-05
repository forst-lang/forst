// Package forstpkg groups and merges Forst sources so a directory can behave like one package.
package forstpkg

import (
	"fmt"
	"os"
	"sort"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// ParseForstFile parses a single .ft file into top-level AST nodes.
func ParseForstFile(log *logrus.Logger, path string) ([]ast.Node, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l := lexer.New(source, path, log)
	tokens := l.Lex()
	psr := parser.New(tokens, path, log)
	return psr.ParseFile()
}

// PackageNameFromNodes returns the declared package name, or "" if missing.
func PackageNameFromNodes(nodes []ast.Node) string {
	for _, node := range nodes {
		if pkgNode, ok := node.(ast.PackageNode); ok {
			return string(pkgNode.Ident.ID)
		}
	}
	return ""
}

// PackageNameOrDefault returns the declared package name, or "main" if missing (Go default).
func PackageNameOrDefault(name string) string {
	if name == "" {
		return "main"
	}
	return name
}

// MergePackageASTs concatenates per-file ASTs in the given order into one node list
// suitable for a single typechecker.CheckTypes pass. Package declarations after the
// first file are skipped (one package clause per compilation unit).
func MergePackageASTs(fileNodes [][]ast.Node) []ast.Node {
	var merged []ast.Node
	for i, nodes := range fileNodes {
		for _, n := range nodes {
			if i > 0 {
				if _, ok := n.(ast.PackageNode); ok {
					continue
				}
			}
			merged = append(merged, n)
		}
	}
	return merged
}

// ParseAndMergePackage parses paths in sorted order and returns merged nodes plus
// per-path AST for per-file metadata (e.g. discovery).
func ParseAndMergePackage(log *logrus.Logger, paths []string) (merged []ast.Node, byPath map[string][]ast.Node, err error) {
	sort.Strings(paths)
	byPath = make(map[string][]ast.Node, len(paths))
	var lists [][]ast.Node
	for _, p := range paths {
		nodes, e := ParseForstFile(log, p)
		if e != nil {
			return nil, nil, fmt.Errorf("parse %s: %w", p, e)
		}
		byPath[p] = nodes
		lists = append(lists, nodes)
	}
	return MergePackageASTs(lists), byPath, nil
}
