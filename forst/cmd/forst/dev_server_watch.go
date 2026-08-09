package main

import (
	"os"
	"path/filepath"
	"time"

	"forst/internal/codegen/layout"
	"forst/internal/devserver"
)

func (s *DevServer) boundaryRoot() string {
	if s.discoverer != nil {
		return s.discoverer.GetRootDir()
	}
	return ""
}

func (s *DevServer) startWatchGenerate() {
	if s.config == nil || !s.config.Dev.EffectiveWatchGenerate() {
		return
	}
	root := s.boundaryRoot()
	if root == "" {
		return
	}
	if err := runGenerateForDev(root, s.config, s.log); err != nil {
		s.log.Warnf("watchGenerate: initial forst generate failed: %v", err)
	}
	stop := make(chan struct{})
	s.watchStop = stop
	go func() {
		_ = devserver.WatchPackageRoot(s.log, root, &s.config.Config, 0, func(_ string) {
			if err := runGenerateForDev(root, s.config, s.log); err != nil {
				s.log.Warnf("watchGenerate: forst generate failed: %v", err)
				return
			}
			s.typesCacheMu.Lock()
			delete(s.typesCache, "types")
			s.lastTypesGen = time.Time{}
			s.typesCacheMu.Unlock()
		}, stop)
	}()
}

func (s *DevServer) stopWatchGenerate() {
	if s.watchStop != nil {
		close(s.watchStop)
		s.watchStop = nil
	}
}

func (s *DevServer) readGeneratedTypesContent() (string, error) {
	root := s.boundaryRoot()
	if root == "" {
		return "", os.ErrNotExist
	}
	if s.config != nil && s.config.Dev.EffectiveWatchGenerate() {
		if err := runGenerateForDev(root, s.config, s.log); err != nil {
			s.log.Warnf("watchGenerate: forst generate failed: %v", err)
		}
	}
	typesPath := filepath.Join(layout.NewRoot(root).ClientDir(), "dist", "types.d.ts")
	data, err := os.ReadFile(typesPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
