package main

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func setupTestLogger(opts *ast.TestLoggerOptions) *logrus.Logger {
	return ast.SetupTestLogger(opts)
}
