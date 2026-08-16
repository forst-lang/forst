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
