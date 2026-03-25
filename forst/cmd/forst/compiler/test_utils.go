package compiler

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func setupTestLogger(opts *ast.TestLoggerOptions) *logrus.Logger {
	return ast.SetupTestLogger(opts)
}
