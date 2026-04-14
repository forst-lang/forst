package compiler

import (
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchFile watches the Forst file for changes and recompiles it.
func (c *Compiler) WatchFile() error {
	if c.Args.PackageRoot != "" {
		return fmt.Errorf("-watch cannot be used with -root; use a single-file compile or omit -watch")
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

func (c *Compiler) runCompiledOutput(code string) error {
	outputPath := c.Args.OutputPath
	if outputPath == "" {
		var err error
		outputPath, err = CreateTempOutputFile(code)
		if err != nil {
			c.log.Error(err)
			os.Exit(1)
		}
	}
	return RunGoProgram(outputPath)
}
