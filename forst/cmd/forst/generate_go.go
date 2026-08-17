package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

type generateGoPlan struct {
	active    bool
	entryPath string
	outPath   string
	root      string
}

func mergeGenerateGoConfig(opts generateOptions, cfg *ForstConfig, boundaryRoot string, targetIsDir bool, target string) ftconfig.GenerateGoConfig {
	merged := cfg.Generate.Go
	if opts.goEntry != "" {
		merged.Entry = opts.goEntry
	}
	if opts.goOut != "" {
		merged.Out = opts.goOut
	}
	if opts.goRoot != "" {
		merged.Root = opts.goRoot
	}
	if !targetIsDir && merged.Entry == "" && merged.Out != "" && strings.HasSuffix(strings.ToLower(target), ".ft") {
		merged.Entry = target
	}
	_ = boundaryRoot
	return merged
}

func resolveGenerateGoPlan(opts generateOptions, cfg *ForstConfig, boundaryRoot string) (generateGoPlan, error) {
	merged := mergeGenerateGoConfig(opts, cfg, boundaryRoot, opts.targetIsDir, opts.target)
	if err := merged.Validate(); err != nil {
		return generateGoPlan{}, fmt.Errorf("generate config: %w", err)
	}
	if !merged.IsConfigured() {
		return generateGoPlan{}, nil
	}

	entry := merged.Entry
	absEntry, err := absPathForGenerate(entry)
	if err != nil {
		return generateGoPlan{}, err
	}
	if filepath.Ext(absEntry) != ".ft" {
		return generateGoPlan{}, fmt.Errorf("generate.go.entry must be a .ft file, got %q", entry)
	}

	outPath := merged.EffectiveGoOut(boundaryRoot)
	if filepath.Ext(outPath) != ".go" {
		return generateGoPlan{}, fmt.Errorf("generate.go.out %q must end with .go", merged.Out)
	}

	return generateGoPlan{
		active:    true,
		entryPath: absEntry,
		outPath:   outPath,
		root:      merged.Root,
	}, nil
}

func shouldSkipClientGenerate(opts generateOptions, cfg *ForstConfig) bool {
	return cfg.Generate.SkipClient || opts.skipClient
}

func runGenerateGoSources(plan generateGoPlan, cfg *ForstConfig, log *logrus.Logger) error {
	if !plan.active {
		return nil
	}
	root := plan.root
	if root != "" {
		absRoot, err := absPathForGenerate(root)
		if err != nil {
			return err
		}
		root = absRoot
	} else if found, err := ftconfig.BoundaryRootFromDir(filepath.Dir(plan.entryPath)); err == nil {
		root = found
	} else {
		root = filepath.Dir(plan.entryPath)
	}
	args := compiler.Args{
		Command:            "generate",
		FilePath:           plan.entryPath,
		OutputPath:         plan.outPath,
		PackageRoot:        root,
		ExportStructFields: cfg.Compiler.ExportStructFields,
		LogLevel:           "error",
	}
	return compiler.EmitGoSources(args, log)
}
