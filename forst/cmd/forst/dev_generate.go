package main

import (
	"forst/internal/devserver"
	"forst/internal/ftconfig"

	"github.com/sirupsen/logrus"
)

// runGenerateForDev regenerates the TypeScript client when watchGenerate is enabled.
func runGenerateForDev(boundaryRoot string, cfg *ForstConfig, log *logrus.Logger) error {
	if cfg == nil || !cfg.Dev.EffectiveWatchGenerate() {
		return nil
	}
	if log != nil {
		log.Debug("watchGenerate: running forst generate")
	}
	return runGenerateOnce(generateOptions{target: boundaryRoot}, cfg, true, log)
}

func afterReloadGenerateHook(cfg *ForstConfig, log *logrus.Logger) func(string, *ftconfig.Config) error {
	return func(boundaryRoot string, _ *ftconfig.Config) error {
		return runGenerateForDev(boundaryRoot, cfg, log)
	}
}

func devRuntimeRunDeps(cfg *ForstConfig, log *logrus.Logger) devserver.RuntimeRunDeps {
	return devserver.RuntimeRunDeps{
		AfterReload: afterReloadGenerateHook(cfg, log),
	}
}
