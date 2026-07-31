package lsp

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type buildMetadata struct {
	version string
	commit  string
	date    string
}

var (
	buildInfoMu sync.RWMutex
	buildInfo   = buildMetadata{
		version: "dev",
		commit:  "unknown",
		date:    "unknown",
	}
)

// SetBuildMetadata sets injected compiler build metadata (version, commit, date).
func SetBuildMetadata(version, commit, date string) {
	buildInfoMu.Lock()
	buildInfo = buildMetadata{version: version, commit: commit, date: date}
	buildInfoMu.Unlock()
}

func buildMetadataSnapshot() buildMetadata {
	buildInfoMu.RLock()
	defer buildInfoMu.RUnlock()
	return buildInfo
}

// BuildInfo returns injected compiler build metadata (version, commit, date).
func BuildInfo() (version, commit, date string) {
	meta := buildMetadataSnapshot()
	return meta.version, meta.commit, meta.date
}

// BuildInfoMap returns build metadata as a JSON-friendly map.
func BuildInfoMap() map[string]string {
	meta := buildMetadataSnapshot()
	return map[string]string{
		"version": meta.version,
		"commit":  meta.commit,
		"date":    meta.date,
	}
}

// LogBuildInfo logs version, commit, and build date at info level (same format as `forst version`).
func LogBuildInfo(log *logrus.Logger) {
	if log == nil {
		return
	}
	meta := buildMetadataSnapshot()
	log.Infof("forst %s %s %s", meta.version, meta.commit, meta.date)
}
