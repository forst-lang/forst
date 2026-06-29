package compiler

import (
	"bytes"
	"io"
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
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	args := ParseArgsFrom([]string{"forst"}, log)
	if args.Command != "" {
		t.Fatalf("expected empty command, got %q", args.Command)
	}
	if !strings.Contains(buf.String(), "Forst Compiler") || !strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("expected usage banner in log, got %q", buf.String())
	}
}

func TestParseArgs_unknownCommand_returnsEmpty(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgsFrom([]string{"forst", "nope", "x.ft"}, log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "Unknown command") {
		t.Fatalf("expected unknown command log, got %q", buf.String())
	}
}

func TestParseArgs_buildWithWatchRejected(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgsFrom([]string{"forst", "build", "-watch", "out.go", "x.ft"}, log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "watch") || !strings.Contains(buf.String(), "build") {
		t.Fatalf("expected watch+build error, got %q", buf.String())
	}
}

func TestParseArgs_watchWithoutOutputRejected(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgsFrom([]string{"forst", "run", "-watch", "x.ft"}, log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "-o") {
		t.Fatalf("expected -o required message, got %q", buf.String())
	}
}

func TestParseArgs_rootWithWatchRejected(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)

	args := ParseArgsFrom([]string{"forst", "run", "-watch", "-o", "out.go", "-root", "/tmp", "x.ft"}, log)
	if args.Command != "" {
		t.Fatalf("expected empty args, got command %q", args.Command)
	}
	if !strings.Contains(buf.String(), "root") || !strings.Contains(buf.String(), "watch") {
		t.Fatalf("expected root+watch error, got %q", buf.String())
	}
}

func TestParseArgs_runSuccess(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := ParseArgsFrom([]string{
		"forst", "run",
		"-loglevel", "debug",
		"-report-memory-usage",
		"-report-phases",
		"-export-struct-fields",
		"-root", "/tmp/pkg",
		"main.ft",
	}, log)
	if args.Command != "run" || args.FilePath != "main.ft" {
		t.Fatalf("got %+v", args)
	}
	if args.LogLevel != "debug" || !args.ReportMemoryUsage || !args.ReportPhases || !args.ExportStructFields {
		t.Fatalf("flags: %+v", args)
	}
	if args.PackageRoot == "" {
		t.Fatal("expected abs package root")
	}
}

func TestParseArgs_buildSuccess(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := ParseArgsFrom([]string{"forst", "build", "-o", "out.go", "main.ft"}, log)
	if args.Command != "build" || args.OutputPath != "out.go" || args.FilePath != "main.ft" {
		t.Fatalf("got %+v", args)
	}
}

func TestParseArgs_runWatchSuccess(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := ParseArgsFrom([]string{"forst", "run", "-watch", "-o", "out.go", "main.ft"}, log)
	if args.Command != "run" || !args.Watch || args.OutputPath != "out.go" {
		t.Fatalf("got %+v", args)
	}
}

func TestParseArgs_missingFileArg(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	args := ParseArgsFrom([]string{"forst", "run"}, log)
	if args.Command != "" {
		t.Fatalf("expected empty, got %+v", args)
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Fatalf("got %q", buf.String())
	}
}
