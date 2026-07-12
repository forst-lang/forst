package devserver

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const defaultWatchDebounce = 100 * time.Millisecond

// WatchPackageRoot watches .ft writes under boundaryRoot and calls onChange (debounced).
func WatchPackageRoot(log *logrus.Logger, boundaryRoot string, cfg *ftconfig.Config, debounce time.Duration, onChange func()) error {
	boundaryRoot = filepath.Clean(boundaryRoot)
	if debounce <= 0 {
		debounce = defaultWatchDebounce
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil && log != nil {
			log.Errorf("close watcher: %v", err)
		}
	}()

	dirs, err := collectWatchDirs(boundaryRoot, cfg)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			return fmt.Errorf("watch %s: %w", dir, err)
		}
	}
	if log != nil {
		log.Infof("Watching Forst sources under %s (%d directories)...", boundaryRoot, len(dirs))
	}

	var debounceTimer *time.Timer
	trigger := func() {
		watchDebounce(&debounceTimer, debounce, onChange)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			if !isWatchedForstFile(event.Name, cfg) {
				continue
			}
			if log != nil {
				log.Infof("Detected change in %s, triggering reload...", event.Name)
			}
			trigger()
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			if log != nil {
				log.Error("Error watching files:", err)
			}
		}
	}
}

func watchDebounce(timer **time.Timer, d time.Duration, fn func()) {
	if *timer != nil {
		(*timer).Stop()
	}
	*timer = time.AfterFunc(d, fn)
}

func isWatchedForstFile(path string, cfg *ftconfig.Config) bool {
	if !strings.HasSuffix(strings.ToLower(path), ".ft") {
		return false
	}
	if cfg == nil {
		return true
	}
	if !cfg.MatchesIncludePatterns(path) {
		return false
	}
	return !cfg.MatchesExcludePatterns(path)
}

func collectWatchDirs(root string, cfg *ftconfig.Config) ([]string, error) {
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("watch root %s: %w", root, err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("watch root is not a directory: %s", root)
	}

	var dirs []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && skipWatchDir(path, cfg) {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})
	return dirs, err
}

func skipWatchDir(absPath string, cfg *ftconfig.Config) bool {
	base := filepath.Base(absPath)
	switch base {
	case "node_modules", ".git", "dist", ".forst":
		return true
	}
	if cfg != nil && cfg.MatchesExcludePatterns(absPath) {
		return true
	}
	return false
}

// RuntimeWatchEnabled reports whether runtime dev should watch and hot-reload.
func RuntimeWatchEnabled(cfg *ftconfig.Config) bool {
	if cfg == nil {
		return false
	}
	return cfg.Dev.Watch || cfg.Dev.HotReload
}

// runningChild wraps a background go run process with injectable stop for tests.
type runningChild struct {
	stop func() error
}

func defaultStartProgram(outputPath, boundaryRoot string) (*runningChild, error) {
	proc, err := compiler.StartGoProgram(outputPath, boundaryRoot)
	if err != nil {
		return nil, err
	}
	return &runningChild{
		stop: func() error { return proc.Stop(compiler.DefaultGoProgramStopGrace()) },
	}, nil
}
