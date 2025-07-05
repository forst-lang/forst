package transformergo

import (
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	return logger
}

func setupTypeChecker(log *logrus.Logger) *typechecker.TypeChecker {
	return typechecker.New(log, false)
}

func setupTransformer(tc *typechecker.TypeChecker, log *logrus.Logger) *Transformer {
	return New(tc, log)
}
