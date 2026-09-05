package invokeserver

import (
	"fmt"
	"os"
	"strings"
	"sync"

	logrus "github.com/sirupsen/logrus"
)

// LogrusLogger adapts logrus to invokeserver.Logger with component=invoke.
type LogrusLogger struct {
	entry *logrus.Entry
}

// NewLogrusLogger wraps log (or stderr logrus when nil) for invoke server logging.
func NewLogrusLogger(log *logrus.Logger) LogrusLogger {
	if log == nil {
		log = defaultInvokeLogrus()
	}
	return LogrusLogger{entry: log.WithField("component", "invoke")}
}

func (l LogrusLogger) Infof(format string, args ...any)  { l.entry.Infof(format, args...) }
func (l LogrusLogger) Warnf(format string, args ...any)  { l.entry.Warnf(format, args...) }
func (l LogrusLogger) Errorf(format string, args ...any) { l.entry.Errorf(format, args...) }
func (l LogrusLogger) Debugf(format string, args ...any) { l.entry.Debugf(format, args...) }

var (
	defaultLogOnce sync.Once
	defaultLog     *logrus.Logger
)

func defaultInvokeLogrus() *logrus.Logger {
	defaultLogOnce.Do(func() {
		defaultLog = logrus.New()
		defaultLog.SetOutput(os.Stderr)
		setInvokeLogLevel(defaultLog, os.Getenv("FORST_LOG_LEVEL"))
	})
	return defaultLog
}

func setInvokeLogLevel(log *logrus.Logger, level string) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		log.SetLevel(logrus.TraceLevel)
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "warn", "warning":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}
}

// DefaultLogger returns logrus-backed invoke logging aligned with forst dev output.
func DefaultLogger() Logger {
	return NewLogrusLogger(nil)
}

// StderrLogger writes plain invoke-prefixed lines (legacy / tests).
type StderrLogger struct{}

func (StderrLogger) Infof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}

func (StderrLogger) Warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}

func (StderrLogger) Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}

func (StderrLogger) Debugf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "invoke: "+format+"\n", args...)
}
