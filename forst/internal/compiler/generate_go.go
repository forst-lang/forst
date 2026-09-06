package compiler

import (
	"fmt"
	"os"
	"path/filepath"

	"forst/internal/codegen/layout"
	"forst/internal/goload"
	"forst/internal/gowork"

	"github.com/sirupsen/logrus"
)

// EmitGoSources transpiles the entry file and writes Go sources to Args.OutputPath.
// External Forst deps are emitted under .forst/overlay/ and listed in .forst/go.work
// replaces. The consumer go.mod is never modified.
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
	if _, _, _, _, _, err := c.CompileWithBridgeRuntime(); err != nil {
		return err
	}
	if err := writeGenerateOverlayGoWork(c, args, log); err != nil {
		return err
	}
	log.Infof("Wrote Go sources beside %s", args.OutputPath)
	return nil
}

func writeGenerateOverlayGoWork(c *Compiler, args Args, log *logrus.Logger) error {
	replaces := c.LastOverlayReplaces()
	if len(replaces) == 0 {
		return nil
	}
	boundary := RunBoundaryRoot(args)
	if boundary == "" {
		return nil
	}
	userMod := goload.FindModuleRoot(boundary)
	if userMod == "" {
		userMod = goload.FindModuleRoot(filepath.Dir(args.FilePath))
	}
	useDirs := make([]string, 0, 1)
	if userMod != "" && !goload.IsForstGoModShim(userMod) {
		useDirs = append(useDirs, userMod)
	}
	if len(useDirs) == 0 {
		// No local module to use; still list overlays so the file documents replaces.
		for _, r := range replaces {
			useDirs = append(useDirs, r.Dir)
		}
	}
	workPath := layout.NewRoot(boundary).GoWork()
	if err := gowork.WriteGoWork(workPath, useDirs, replaces); err != nil {
		return fmt.Errorf("write overlay go.work: %w", err)
	}
	if log != nil {
		log.Infof("Wrote Forst dependency overlays under %s (go.work replaces; user go.mod unchanged)", filepath.Join(boundary, ".forst", "overlay"))
	}
	return nil
}

func ensureGoSourceOutputPath(outputPath string) error {
	if !stringsHasGoExt(outputPath) {
		return fmt.Errorf("Go source output path %q must end with .go", outputPath)
	}
	return nil
}

func stringsHasGoExt(path string) bool {
	return filepath.Ext(path) == ".go"
}
