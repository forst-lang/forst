package forstpkg

import (
	"context"
	"fmt"
	"sort"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const defaultParseParallelism = 4

// ParseFilesLenientParallel parses paths concurrently, skipping files that fail to parse.
func ParseFilesLenientParallel(log *logrus.Logger, paths []string) map[string][]ast.Node {
	byPath, err := parseFilesParallel(log, paths, len(paths))
	if err == nil {
		return byPath
	}
	out := make(map[string][]ast.Node)
	for _, path := range paths {
		nodes, perr := ParseForstFile(log, path)
		if perr == nil {
			out[path] = nodes
		}
	}
	return out
}

// ParseFilesParallel parses .ft paths concurrently with deterministic ordering.
func ParseFilesParallel(log *logrus.Logger, paths []string) (map[string][]ast.Node, error) {
	return parseFilesParallel(log, paths, len(paths))
}

func parseLimit(n int) int {
	if n <= 0 {
		return defaultParseParallelism
	}
	if n < defaultParseParallelism {
		return n
	}
	return defaultParseParallelism
}

// parseFilesParallel lexes and parses each path; results are keyed by path.
func parseFilesParallel(log *logrus.Logger, paths []string, limit int) (map[string][]ast.Node, error) {
	if len(paths) == 0 {
		return map[string][]ast.Node{}, nil
	}
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	if len(sorted) == 1 {
		nodes, err := ParseForstFile(log, sorted[0])
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", sorted[0], err)
		}
		return map[string][]ast.Node{sorted[0]: nodes}, nil
	}
	results := make([][]ast.Node, len(sorted))
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(parseLimit(limit))
	for i := range sorted {
		i := i
		g.Go(func() error {
			nodes, err := ParseForstFile(log, sorted[i])
			if err != nil {
				return fmt.Errorf("parse %s: %w", sorted[i], err)
			}
			results[i] = nodes
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	byPath := make(map[string][]ast.Node, len(sorted))
	for i, path := range sorted {
		byPath[path] = results[i]
	}
	return byPath, nil
}
