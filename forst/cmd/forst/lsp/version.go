package lsp

import "github.com/sirupsen/logrus"

// BuildInfo returns injected compiler build metadata (version, commit, date).
func BuildInfo() (version, commit, date string) {
	return Version, Commit, Date
}

// BuildInfoMap returns build metadata as a JSON-friendly map.
func BuildInfoMap() map[string]string {
	return map[string]string{
		"version": Version,
		"commit":  Commit,
		"date":    Date,
	}
}

// LogBuildInfo logs version, commit, and build date at info level (same format as `forst version`).
func LogBuildInfo(log *logrus.Logger) {
	if log == nil {
		return
	}
	log.Infof("forst %s %s %s", Version, Commit, Date)
}
