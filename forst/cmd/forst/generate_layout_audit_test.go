package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/ftconfig"
	"forst/internal/forstpkg"
	transformerts "forst/internal/transformer/ts"
)

// generateBoundaries lists ftconfig roots whose .ft layout must pass stem and Go layout checks in CI.
var generateBoundaries = []string{
	"examples/in/tictactoe",
	"examples/in/generate-effect",
	"examples/in/rfc/bridge-interop/remix-serve",
	"examples/in/rfc/bridge-interop/multi-package-dev",
	"examples/in/rfc/sidecar",
}

func TestGenerateBoundaries_stemAndGoLayoutValid(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	for _, rel := range generateBoundaries {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			boundary := filepath.Join(repoRoot, rel)
			cfgPath := filepath.Join(boundary, "ftconfig.json")
			if _, err := os.Stat(cfgPath); err != nil {
				t.Skip("no ftconfig.json")
			}
			cfg, err := ftconfig.Load(cfgPath)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}
			paths, err := cfg.FindForstFiles(boundary)
			if err != nil {
				t.Fatalf("find files: %v", err)
			}
			if len(paths) == 0 {
				t.Fatal("expected at least one .ft file")
			}
			if err := transformerts.ValidateDiscoveredFileStems(paths, false, nil); err != nil {
				t.Fatalf("stem layout: %v", err)
			}
			byPackage := make(map[string][]string)
			for _, p := range paths {
				if strings.HasSuffix(p, "_test.ft") {
					continue
				}
				nodes, err := forstpkg.ParseForstFile(nil, p)
				if err != nil {
					t.Fatalf("parse %s: %v", p, err)
				}
				pkg := forstpkg.PackageNameOrDefault(forstpkg.PackageNameFromNodes(nodes))
				byPackage[pkg] = append(byPackage[pkg], p)
			}
			if err := forstpkg.ValidateGoPackageLayout(byPackage); err != nil {
				t.Fatalf("go package layout: %v", err)
			}
		})
	}
}
