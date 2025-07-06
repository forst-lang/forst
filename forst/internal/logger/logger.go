package logger

import (
	logrus "github.com/sirupsen/logrus"
)

// New creates a new logger with the default settings
func New() *logrus.Logger {
	return NewWithLevel(logrus.DebugLevel)
}

// NewWithLevel creates a new logger with the specified level
func NewWithLevel(level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.000",
	})
	return logger
}
