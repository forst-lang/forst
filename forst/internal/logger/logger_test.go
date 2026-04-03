package logger

import (
	"testing"

	logrus "github.com/sirupsen/logrus"
)

func TestNew_is_debug_level(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("nil logger")
	}
	if l.GetLevel() != logrus.DebugLevel {
		t.Fatalf("GetLevel() = %v, want DebugLevel", l.GetLevel())
	}
}

func TestNewWithLevel_sets_level(t *testing.T) {
	l := NewWithLevel(logrus.WarnLevel)
	if l == nil {
		t.Fatal("NewWithLevel returned nil logger")
	}
	if l.GetLevel() != logrus.WarnLevel {
		t.Fatalf("GetLevel() = %v, want WarnLevel", l.GetLevel())
	}
}
