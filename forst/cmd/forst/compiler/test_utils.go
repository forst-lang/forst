package compiler

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	return ast.SetupTestLogger()
}
