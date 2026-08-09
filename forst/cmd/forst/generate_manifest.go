package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"forst/internal/ftconfig"
	transformerts "forst/internal/transformer/ts"

	"github.com/sirupsen/logrus"
)

// generateManifestEntry describes one exported function in the manifest.
type generateManifestEntry struct {
	Package    string `json:"package"`
	Function   string `json:"function"`
	Streaming  bool   `json:"streaming,omitempty"`
	Omitted    bool   `json:"omitted,omitempty"`
	OmitReason string `json:"omitReason,omitempty"`
}

// generateManifest is the machine-readable output for forst generate --list --json.
type generateManifest struct {
	PackageName string                  `json:"packageName"`
	OutDir      string                  `json:"outDir"`
	Packages    []string                `json:"packages"`
	Functions   []generateManifestEntry `json:"functions"`
	Omitted     []generateManifestEntry `json:"omitted,omitempty"`
}

func printGenerateManifest(boundaryRoot string, genCfg ftconfig.GenerateConfig, outputs []*transformerts.TypeScriptOutput, _ *logrus.Logger) error {
	manifest := generateManifest{
		PackageName: genCfg.PackageName,
		OutDir:      genCfg.EffectiveOutDir(boundaryRoot),
	}

	pkgSet := map[string]struct{}{}
	for _, out := range outputs {
		if out == nil || out.PackageName == "" {
			continue
		}
		pkgSet[out.PackageName] = struct{}{}
		for _, fn := range out.Functions {
			if fn.Name == "" {
				continue
			}
			entry := generateManifestEntry{
				Package:  out.PackageName,
				Function: fn.Name,
			}
			if fn.StreamingRowType != "" {
				entry.Streaming = true
			}
			manifest.Functions = append(manifest.Functions, entry)
		}
	}
	for pkg := range pkgSet {
		manifest.Packages = append(manifest.Packages, pkg)
	}
	sort.Strings(manifest.Packages)
	sort.Slice(manifest.Functions, func(i, j int) bool {
		a, b := manifest.Functions[i], manifest.Functions[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		return a.Function < b.Function
	})

	enc := json.NewEncoder(generateReportWriter)
	enc.SetIndent("", "  ")
	if err := enc.Encode(manifest); err != nil {
		return err
	}
	_, err := fmt.Fprintln(generateReportWriter)
	return err
}
