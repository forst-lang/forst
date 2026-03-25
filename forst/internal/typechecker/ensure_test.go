package typechecker

import (
	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func setupTestLogger(opts *ast.TestLoggerOptions) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	if opts != nil {
		logger.SetLevel(opts.ForceLevel)
	}

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	return logger
}
