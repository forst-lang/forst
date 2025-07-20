package main

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	return ast.SetupTestLogger()
}
