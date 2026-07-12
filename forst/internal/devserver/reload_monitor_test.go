package devserver

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestRunningChild_stopForReload_suppressesExitAlarm(t *testing.T) {
	exited := make(chan error, 1)
	exited <- fmt.Errorf("generated program exited with code -1 (see stderr above for details)")

	child := &runningChild{
		stop:   func() error { return nil },
		exited: exited,
	}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	child.stopForReload(log, t.TempDir())
	time.Sleep(50 * time.Millisecond)

	if strings.Contains(buf.String(), "Generated program exited") {
		t.Fatalf("unexpected exit alarm: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "Stopped previous build for reload") {
		t.Fatalf("expected debug stop message, got: %s", buf.String())
	}
}

func TestHostModeEnabled(t *testing.T) {
	dir := t.TempDir()
	if hostModeEnabled(dir) {
		t.Fatal("expected false without ftconfig")
	}
	if err := os.WriteFile(filepath.Join(dir, "ftconfig.json"), []byte(`{"node":{"hostMode":true,"args":["x"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hostModeEnabled(dir) {
		t.Fatal("expected true when hostMode set")
	}
}

func TestMonitorChildExit_unexpectedCrashStillLogs(t *testing.T) {
	exited := make(chan error, 1)
	exited <- fmt.Errorf("generated program exited with code 1")

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	monitorChildExit(log, &runningChild{exited: exited})
	time.Sleep(50 * time.Millisecond)

	if !strings.Contains(buf.String(), "Generated program exited") {
		t.Fatalf("expected crash warn, got: %s", buf.String())
	}
}
