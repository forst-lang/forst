package compiler

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestPrintVersion_smoke(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	printVersion(log)
}

func TestPrintUsage_smoke(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	log.SetOutput(io.Discard)
	printUsage(log)
}

func TestParseArgs_noSubcommand_printsUsageAndReturnsEmpty(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"forst"}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	args := ParseArgs(log)
	if args.Command != "" {
		t.Fatalf("expected empty command, got %q", args.Command)
	}
	if !strings.Contains(buf.String(), "Forst Compiler") || !strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("expected usage banner in log, got %q", buf.String())
	}
}

func TestParseArgs_unknownCommand_returnsEmpty(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"forst", "nope", "x.ft"}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgs(log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "Unknown command") {
		t.Fatalf("expected unknown command log, got %q", buf.String())
	}
}

func TestParseArgs_buildWithWatchRejected(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"forst", "build", "-watch", "out.go", "x.ft"}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgs(log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "watch") || !strings.Contains(buf.String(), "build") {
		t.Fatalf("expected watch+build error, got %q", buf.String())
	}
}

func TestParseArgs_watchWithoutOutputRejected(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"forst", "run", "-watch", "x.ft"}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgs(log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "-o") {
		t.Fatalf("expected -o required message, got %q", buf.String())
	}
}

func TestParseArgs_rootWithWatchRejected(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"forst", "run", "-watch", "-o", "out.go", "-root", "/tmp", "x.ft"}

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgs(log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "root") || !strings.Contains(buf.String(), "watch") {
		t.Fatalf("expected root+watch error, got %q", buf.String())
	}
}
