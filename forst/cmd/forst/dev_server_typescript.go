package main

import (
	"forst/internal/discovery"
	transformerts "forst/internal/transformer/ts"

	logrus "github.com/sirupsen/logrus"
)

// TypeScriptGenerator handles TypeScript type generation.
type TypeScriptGenerator struct {
	log *logrus.Logger
}

// NewTypeScriptGenerator creates a new TypeScript generator.
func NewTypeScriptGenerator(log *logrus.Logger) *TypeScriptGenerator {
	return &TypeScriptGenerator{
		log: log,
	}
}

// GenerateTypesForFunctions generates TypeScript types for discovered functions.
func (tg *TypeScriptGenerator) GenerateTypesForFunctions(functions map[string]map[string]discovery.FunctionInfo, _ string) (string, error) {
	// Collect all Forst files that contain discovered functions.
	filePaths := make(map[string]bool)
	for _, pkgFuncs := range functions {
		for _, fn := range pkgFuncs {
			filePaths[fn.FilePath] = true
		}
	}

	var outputs []*transformerts.TypeScriptOutput
	for filePath := range filePaths {
		out, err := transformerts.TransformForstFileFromPath(filePath, tg.log, transformerts.TransformForstFileOptions{
			RelaxedTypecheck: true,
		})
		if err != nil {
			tg.log.Warnf("Failed to generate types for %s: %v", filePath, err)
			continue
		}
		outputs = append(outputs, out)
	}

	if len(outputs) == 0 {
		return (&transformerts.TypeScriptOutput{}).GenerateTypesFile(), nil
	}

	merged, err := transformerts.MergeTypeScriptOutputs(outputs)
	if err != nil {
		return "", err
	}

	return merged.GenerateTypesFile(), nil
}

// generateTypesForFile generates TypeScript types for a single Forst file.
func (tg *TypeScriptGenerator) generateTypesForFile(filePath string) ([]string, []transformerts.FunctionSignature, string, error) {
	out, err := transformerts.TransformForstFileFromPath(filePath, tg.log, transformerts.TransformForstFileOptions{
		RelaxedTypecheck: true,
	})
	if err != nil {
		return nil, nil, "", err
	}
	return out.Types, out.Functions, out.PackageName, nil
}
