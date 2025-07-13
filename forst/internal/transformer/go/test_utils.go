package transformergo

import (
	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	return ast.SetupTestLogger()
}

func setupTypeChecker(log *logrus.Logger) *typechecker.TypeChecker {
	return typechecker.New(log, false)
}

func setupTransformer(tc *typechecker.TypeChecker, log *logrus.Logger) *Transformer {
	return New(tc, log)
}
