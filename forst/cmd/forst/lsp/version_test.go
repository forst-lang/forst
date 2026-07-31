package lsp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogBuildInfo_logsVersionCommitDate(t *testing.T) {
	t.Parallel()
	origVersion, origCommit, origDate := BuildInfo()
	SetBuildMetadata("1.2.3", "abc123", "2026-07-08")
	t.Cleanup(func() {
		SetBuildMetadata(origVersion, origCommit, origDate)
	})

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	LogBuildInfo(log)
	got := buf.String()
	for _, want := range []string{"forst", "1.2.3", "abc123", "2026-07-08"} {
		if !strings.Contains(got, want) {
			t.Fatalf("log %q missing %q", got, want)
		}
	}
}

func TestBuildInfoMap(t *testing.T) {
	t.Parallel()
	origVersion, origCommit, origDate := BuildInfo()
	SetBuildMetadata("v", "c", "d")
	t.Cleanup(func() {
		SetBuildMetadata(origVersion, origCommit, origDate)
	})
	m := BuildInfoMap()
	if m["version"] != "v" || m["commit"] != "c" || m["date"] != "d" {
		t.Fatalf("BuildInfoMap() = %#v", m)
	}
}
