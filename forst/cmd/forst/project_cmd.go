package main

import (
	"flag"
	"fmt"
	"os"

	"forst/internal/project"

	"github.com/sirupsen/logrus"
)

func runProjectInfo(args []string, log *logrus.Logger) int {
	fs := flag.NewFlagSet("project", flag.ExitOnError)
	root := fs.String("root", ".", "ftconfig boundary root")
	_ = fs.Parse(args)

	proj, err := project.Open(log, project.OpenOpts{BoundaryRoot: *root, Cwd: *root})
	if err != nil {
		log.Error(err)
		return 1
	}
	fmt.Printf("boundary: %s\n", proj.BoundaryRoot)
	fmt.Printf("moduleRoot: %s\n", proj.ModuleRoot)
	fmt.Printf("modulePath: %s\n", proj.ModulePath)
	fmt.Printf("devProfile: %s\n", proj.DevProfile())
	fmt.Printf("packages: %v\n", proj.ForstPackages())
	runnable, _ := proj.RunnableFunctions()
	fmt.Printf("runnableExports: %d\n", len(runnable))
	return 0
}

func runClean(args []string, log *logrus.Logger) int {
	fs := flag.NewFlagSet("clean", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", ".", "ftconfig boundary root")
	dryRun := fs.Bool("dry-run", false, "print paths that would be removed without deleting")
	verbose := fs.Bool("verbose", false, "log removed paths")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	result, err := project.CleanGenerated(*root, *dryRun)
	if err != nil {
		log.Error(err)
		return 1
	}
	if len(result.Removed) == 0 {
		log.Infof("clean: nothing to remove under %s", result.BoundaryRoot)
		return 0
	}
	for _, path := range result.Removed {
		if *dryRun {
			log.Infof("clean: would remove %s", path)
		} else if *verbose {
			log.Infof("clean: removed %s", path)
		}
	}
	if !*dryRun && !*verbose {
		log.Infof("clean: removed %s", result.Removed[0])
	}
	return 0
}
