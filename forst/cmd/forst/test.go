package main

import (
	"os"
	"path/filepath"

	"forst/internal/goload"
	"forst/internal/testrunner"

	logrus "github.com/sirupsen/logrus"
)

func runTestCommand(args []string, log *logrus.Logger) int {
	root, err := os.Getwd()
	if err != nil {
		log.Error(err)
		return 2
	}
	paths, goTestArgs := testrunner.ParseCLIArgs(args)
	for _, p := range paths {
		if p == "" || p == "." {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			log.Error(err)
			return 2
		}
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			root = goload.FindModuleRoot(abs)
			break
		}
	}
	root = goload.FindModuleRoot(root)

	code, err := testrunner.Run(testrunner.Options{
		ModuleRoot: root,
		Paths:      paths,
		GoTestArgs: goTestArgs,
		Log:        log,
	})
	if err != nil {
		log.Error(err)
		if code == 0 {
			return 2
		}
		return code
	}
	return code
}
