package main

import (
	"forst/internal/semantic"

	"github.com/sirupsen/logrus"
)

func buildSemanticSnapshot(forstFiles []string, boundaryRoot string, log *logrus.Logger) (*semantic.GenerateRequest, error) {
	return semantic.BuildSnapshot(forstFiles, boundaryRoot, log)
}
