package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"forst/internal/devserver"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

// generateWatchDebounce is the delay before regenerating after a .ft change.
// Tests may lower this for faster feedback.
var generateWatchDebounce = 100 * time.Millisecond

// watchPackageRootFn is the package-root watcher used by --watch (tests may replace it).
var watchPackageRootFn = devserver.WatchPackageRoot

// generateWatchStopHook, when non-nil, is closed by tests to end the watch loop early.
// Production uses OS signals instead.
var generateWatchStopHook chan struct{}

// watchGenerate runs generate once, then regenerates on debounced .ft changes.
func watchGenerate(opts generateOptions, cfg *ForstConfig, isDir bool, log *logrus.Logger) error {
	if err := runGenerateOnce(opts, cfg, isDir, log); err != nil {
		return err
	}

	_, outputDir, err := discoverForstFilesForGenerate(cfg, opts.target, isDir)
	if err != nil {
		return err
	}

	boundaryRoot := outputDir
	if root, rootErr := ftconfig.BoundaryRootFromDir(outputDir); rootErr == nil {
		boundaryRoot = root
	}

	stop := make(chan struct{})
	if generateWatchStopHook != nil {
		go func() {
			<-generateWatchStopHook
			close(stop)
		}()
	} else {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			signal.Stop(sigCh)
			close(stop)
		}()
	}

	log.WithFields(logrus.Fields{
		"debounce": generateWatchDebounce.String(),
		"root":     boundaryRoot,
	}).Info("Watching Forst sources for generate")

	return watchPackageRootFn(log, boundaryRoot, &cfg.Config, generateWatchDebounce, func(changedPath string) {
		log.WithFields(logrus.Fields{
			"path": changedPath,
		}).Info("Regenerating TypeScript client after .ft change")
		if err := runGenerateOnce(opts, cfg, isDir, log); err != nil {
			log.WithError(err).Error("generate failed during watch; keeping previous output")
		}
	}, stop)
}
