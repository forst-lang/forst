package main

import (
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
	cwd, err := runTestCommandGetwd()
	if err != nil {
		log.Error(err)
		return 2
	}
	paths, goTestArgs := testrunner.ParseCLIArgs(args)
	root := cwd
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
			root = modRoot
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
	root = goload.FindModuleRoot(root)

	code, err := runTestCommandRunner(testrunner.Options{
		ModuleRoot: root,
		Paths:      paths,
		GoTestArgs: goTestArgs,
		Log:        log,
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
