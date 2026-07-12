package devserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"forst/internal/compiler"
	"forst/internal/ftconfig"
	"forst/nodert"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const defaultWatchDebounce = 100 * time.Millisecond

// WatchPackageRoot watches .ft writes under boundaryRoot and calls onChange (debounced).
// onChange receives the changed file path when available.
// When stop is closed, watching ends and returns nil.
func WatchPackageRoot(log *logrus.Logger, boundaryRoot string, cfg *ftconfig.Config, debounce time.Duration, onChange func(changedPath string), stop <-chan struct{}) error {
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
	if len(dirs) == 0 {
		return fmt.Errorf("no directories to watch under %s", boundaryRoot)
	}
	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			return fmt.Errorf("watch %s: %w", dir, err)
		}
	}
	if log != nil {
		log.Infof("Watching %d Forst source directories under %s...", len(dirs), boundaryRoot)
	}

	var debounceTimer *time.Timer
	var pendingPath string
	trigger := func(changedPath string) {
		if changedPath != "" {
			pendingPath = changedPath
		}
		path := pendingPath
		watchDebounce(&debounceTimer, debounce, func() {
			onChange(path)
		})
	}

	for {
		select {
		case <-stop:
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !isRelevantWatchOp(event.Op) {
				continue
			}
			if !isWatchedForstFile(event.Name, cfg) {
				continue
			}
			if log != nil {
				log.Debugf("Detected change in %s", event.Name)
			}
			trigger(event.Name)
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

func isRelevantWatchOp(op fsnotify.Op) bool {
	return op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0
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

	effective := cfg
	if effective == nil {
		effective = ftconfig.Default()
	}
	files, err := effective.FindForstFiles(root)
	if err != nil {
		return nil, fmt.Errorf("find Forst files under %s: %w", root, err)
	}

	seen := make(map[string]struct{})
	addDir := func(dir string) {
		dir = filepath.Clean(dir)
		if !dirUnderRoot(dir, root) {
			return
		}
		seen[dir] = struct{}{}
	}
	addDir(root)
	for _, file := range files {
		dir := filepath.Dir(filepath.Clean(file))
		for {
			addDir(dir)
			if dir == root {
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs, nil
}

func dirUnderRoot(dir, root string) bool {
	dir = filepath.Clean(dir)
	root = filepath.Clean(root)
	if dir == root {
		return true
	}
	rel, err := filepath.Rel(root, dir)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
	stop            func() error
	exited          <-chan error
	intentionalStop atomic.Bool
	pid             int
}

func defaultStartProgram(outputPath, boundaryRoot string) (*runningChild, error) {
	var proc *compiler.GoProgramProcess
	var err error
	if compiler.IsDevBinPath(outputPath, boundaryRoot) {
		proc, err = compiler.ExecBuiltProgram(outputPath, boundaryRoot)
	} else {
		proc, err = compiler.StartGoProgram(outputPath, boundaryRoot)
	}
	if err != nil {
		return nil, err
	}
	return &runningChild{
		stop: func() error {
			return proc.Stop(compiler.ReloadStopGrace(), compiler.StopOpts{})
		},
		exited: proc.Done(),
		pid:    proc.PID(),
	}, nil
}

func (c *runningChild) stopForReload(log *logrus.Logger, boundaryRoot string) {
	if c == nil {
		return
	}
	c.intentionalStop.Store(true)
	_ = c.stop()
	_ = ClearGoChildPID(boundaryRoot)
	if log != nil {
		if os.Getenv(nodert.EnvNodeAttachOnly) == "1" {
			log.Debug("Stopped previous build for reload (attach-only host preserved)")
		} else {
			log.Debug("Stopped previous build for reload")
		}
	}
	go drainChildExit(c)
}

func drainChildExit(child *runningChild) {
	if child == nil || child.exited == nil {
		return
	}
	<-child.exited
}

func monitorChildExit(log *logrus.Logger, child *runningChild) {
	if log == nil || child == nil || child.exited == nil {
		return
	}
	go func(c *runningChild) {
		err := <-c.exited
		if err != nil && !c.intentionalStop.Load() {
			log.Error(err)
			log.Warn("Generated program exited; fix errors and save a .ft file to reload")
		}
	}(child)
}

func hostModeEnabled(boundaryRoot string) bool {
	if boundaryRoot == "" {
		return false
	}
	cfg, err := ftconfig.LoadFromDir(boundaryRoot)
	if err != nil {
		return false
	}
	return cfg.Node.HostMode
}
