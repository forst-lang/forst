package main

import (
	"flag"
	"os"
	"path/filepath"

	"forst/internal/goload"
	"forst/internal/testrunner"

	logrus "github.com/sirupsen/logrus"
)

var (
	runTestCommandGetwd    = os.Getwd
	runTestCommandPathAbs  = filepath.Abs
	runTestCommandPathRel  = filepath.Rel
	runTestCommandRunner   = testrunner.Run
)

func runTestCommand(args []string, log *logrus.Logger) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	root := fs.String("root", "", "ftconfig boundary root (default: inferred from paths or cwd)")
	exportStructFields := fs.Bool("export-struct-fields", false, "emit json-tagged exported struct fields")
	if err := fs.Parse(args); err != nil {
		log.Error(err)
		return 2
	}
	args = fs.Args()

	cwd, err := runTestCommandGetwd()
	if err != nil {
		log.Error(err)
		return 2
	}
	paths, goTestArgs := testrunner.ParseCLIArgs(args)
	boundary := *root
	if boundary == "" {
		boundary = cwd
	}
	moduleRoot := boundary
	for _, p := range paths {
		if p == "" {
			continue
		}
		candidate := p
		if p == "." {
			candidate = cwd
		}
		abs, err := runTestCommandPathAbs(candidate)
		if err != nil {
			log.Error(err)
			return 2
		}
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			modRoot, err := goload.ModuleRootWithGoMod(abs)
			if err != nil {
				log.Error(err)
				return 2
			}
			moduleRoot = modRoot
			rel, err := runTestCommandPathRel(modRoot, abs)
			if err != nil {
				log.Error(err)
				return 2
			}
			if rel == "." {
				paths = []string{"."}
			} else {
				paths = []string{filepath.ToSlash(rel)}
			}
			break
		}
	}
	moduleRoot = goload.FindModuleRoot(moduleRoot)
	if *root != "" {
		absRoot, err := runTestCommandPathAbs(*root)
		if err != nil {
			log.Error(err)
			return 2
		}
		boundary = absRoot
	} else {
		boundary = moduleRoot
	}

	code, err := runTestCommandRunner(testrunner.Options{
		ModuleRoot:         moduleRoot,
		BoundaryRoot:       boundary,
		Paths:              paths,
		GoTestArgs:         goTestArgs,
		Log:                log,
		ExportStructFields: *exportStructFields,
	})
	if err != nil {
		log.Error(err)
		if code == testrunner.ExitSuccess {
			return testrunner.ExitError.Int()
		}
		return code.Int()
	}
	return code.Int()
}
