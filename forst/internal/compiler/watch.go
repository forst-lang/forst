package compiler

import (
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
)

var createTempOutputFileForWatch = CreateTempOutputFiles
var runGoProgramForWatch = RunGoProgram

// WatchFile watches the Forst file for changes and recompiles it.
func (c *Compiler) WatchFile() error {
	if err := c.validateWatchConfig(); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("error creating watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			c.log.Errorf("Error closing watcher: %v", err)
		}
	}()

	if err := watcher.Add(c.Args.FilePath); err != nil {
		return fmt.Errorf("error watching file: %v", err)
	}

	c.log.Infof("Watching %s for changes...", c.Args.FilePath)
	c.compileAndRunOnce()

	var debounceTimer *time.Timer
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			watchDebounce(&debounceTimer, 100*time.Millisecond, func() {
				c.log.Info("File changed, recompiling...")
				c.compileAndRunOnce()
			})
		case err := <-watcher.Errors:
			c.log.Error("Error watching file:", err)
		}
	}
}

func watchDebounce(timer **time.Timer, d time.Duration, fn func()) {
	if *timer != nil {
		(*timer).Stop()
	}
	*timer = time.AfterFunc(d, fn)
}

func (c *Compiler) validateWatchConfig() error {
	if c.Args.PackageRoot != "" {
		return fmt.Errorf("-watch cannot be used with -root; use a single-file compile or omit -watch")
	}
	return nil
}

func (c *Compiler) compileAndRunOnce() {
	mainCode, nodeRuntimeCode, invokeServerCode, extraPkgs, extraImports, err := c.CompileWithNodeRuntime()
	if err != nil {
		c.log.Error(err)
		c.log.Warn("Not running program because of errors during compilation")
		return
	}
	if err := c.runCompiledOutput(mainCode, nodeRuntimeCode, invokeServerCode, extraPkgs, extraImports); err != nil {
		c.log.Error(err)
	}
}

func (c *Compiler) resolveOutputPathForRun(mainCode, nodeRuntimeCode, invokeServerCode string, extra map[string]string, extraImports map[string]string) (string, error) {
	if c.Args.OutputPath != "" {
		return c.Args.OutputPath, nil
	}
	return createTempOutputFileForWatch(mainCode, nodeRuntimeCode, invokeServerCode, extra, extraImports, RunBoundaryRoot(c.Args))
}

func (c *Compiler) runCompiledOutput(mainCode, nodeRuntimeCode, invokeServerCode string, extra map[string]string, extraImports map[string]string) error {
	outputPath, err := c.resolveOutputPathForRun(mainCode, nodeRuntimeCode, invokeServerCode, extra, extraImports)
	if err != nil {
		return err
	}
	return runGoProgramForWatch(outputPath, RunBoundaryRoot(c.Args))
}
