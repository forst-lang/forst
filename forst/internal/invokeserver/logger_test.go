package invokeserver

import (
	"bytes"
	"strings"
	"testing"

	logrus "github.com/sirupsen/logrus"
)

func TestLogrusLogger_includesComponentInvoke(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	l := NewLogrusLogger(log)
	l.Debugf("http %s %s %d %s", "GET", "/health", 200, "0s")

	out := buf.String()
	if !strings.Contains(out, "component=invoke") {
		t.Fatalf("expected component=invoke in log output, got %q", out)
	}
	if !strings.Contains(out, "GET /health") {
		t.Fatalf("expected request line in log output, got %q", out)
	}
	if strings.HasPrefix(out, "invoke:") {
		t.Fatalf("legacy invoke: prefix should not appear, got %q", out)
	}
}

func TestDefaultLogger_usesLogrus(t *testing.T) {
	l := DefaultLogger()
	if _, ok := l.(LogrusLogger); !ok {
		t.Fatalf("DefaultLogger should return LogrusLogger, got %T", l)
	}
}

func TestSetInvokeLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		envLevel string
		want     logrus.Level
	}{
		{name: "trace", envLevel: "trace", want: logrus.TraceLevel},
		{name: "debug", envLevel: "debug", want: logrus.DebugLevel},
		{name: "warn", envLevel: "warn", want: logrus.WarnLevel},
		{name: "warning", envLevel: "warning", want: logrus.WarnLevel},
		{name: "error", envLevel: "error", want: logrus.ErrorLevel},
		{name: "invalid", envLevel: "not-a-level", want: logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FORST_LOG_LEVEL", tt.envLevel)
			log := logrus.New()
			setInvokeLogLevel(log, tt.envLevel)
			if got := log.GetLevel(); got != tt.want {
				t.Fatalf("GetLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
