package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// EmitGoSources transpiles the entry file and writes Go sources to Args.OutputPath.
func EmitGoSources(args Args, log *logrus.Logger) error {
	if args.FilePath == "" {
		return fmt.Errorf("Go source generation requires an entry .ft file")
	}
	if args.OutputPath == "" {
		return fmt.Errorf("Go source generation requires -o <path.go>")
	}
	if err := ensureGoSourceOutputPath(args.OutputPath); err != nil {
		return err
	}
	if dir := filepath.Dir(args.OutputPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create Go output directory: %w", err)
		}
	}
	c := New(args, log)
	if _, _, _, _, _, err := c.CompileWithNodeRuntime(); err != nil {
		return err
	}
	log.Infof("Wrote Go sources beside %s", args.OutputPath)
	return nil
}

func ensureGoSourceOutputPath(outputPath string) error {
	if filepath.IsAbs(outputPath) {
		return nil
	}
	if !stringsHasGoExt(outputPath) {
		return fmt.Errorf("Go source output path %q must end with .go", outputPath)
	}
	return nil
}

func stringsHasGoExt(path string) bool {
	return filepath.Ext(path) == ".go"
}
