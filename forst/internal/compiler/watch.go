package compiler

import (
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
)

var createTempOutputFileForWatch = CreateTempOutputFile
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
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
				c.log.Info("File changed, recompiling...")
				c.compileAndRunOnce()
			})
		case err := <-watcher.Errors:
			c.log.Error("Error watching file:", err)
		}
	}
}

func (c *Compiler) validateWatchConfig() error {
	if c.Args.PackageRoot != "" {
		return fmt.Errorf("-watch cannot be used with -root; use a single-file compile or omit -watch")
	}
	return nil
}

func (c *Compiler) compileAndRunOnce() {
	code, err := c.CompileFile()
	if err != nil {
		c.log.Error(err)
		c.log.Warn("Not running program because of errors during compilation")
		return
	}
	if err := c.runCompiledOutput(*code); err != nil {
		c.log.Error(err)
	}
}

func (c *Compiler) resolveOutputPathForRun(code string) (string, error) {
	if c.Args.OutputPath != "" {
		return c.Args.OutputPath, nil
	}
	return createTempOutputFileForWatch(code)
}

func (c *Compiler) runCompiledOutput(code string) error {
	outputPath, err := c.resolveOutputPathForRun(code)
	if err != nil {
		return err
	}
	return runGoProgramForWatch(outputPath)
}
