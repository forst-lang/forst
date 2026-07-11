package compiler

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseArgsFrom_helpExits(t *testing.T) {
	var exitCode int
	orig := osExit
	t.Cleanup(func() { osExit = orig })
	osExit = func(code int) { exitCode = code; panic("exit") }

	log := logrus.New()
	log.SetOutput(io.Discard)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected exit panic")
		}
	}()
	_ = ParseArgsFrom([]string{"forst", "--help"}, log)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d", exitCode)
	}
}

func TestParseArgsFrom_versionExits(t *testing.T) {
	var exitCode int
	orig := osExit
	t.Cleanup(func() { osExit = orig })
	osExit = func(code int) { exitCode = code; panic("exit") }

	log := logrus.New()
	log.SetOutput(io.Discard)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected exit panic")
		}
	}()
	_ = ParseArgsFrom([]string{"forst", "-v"}, log)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d", exitCode)
	}
}

func TestParseArgsFrom_commandHelpFlagExits(t *testing.T) {
	var exitCode int
	orig := osExit
	t.Cleanup(func() { osExit = orig })
	osExit = func(code int) { exitCode = code; panic("exit") }

	log := logrus.New()
	log.SetOutput(io.Discard)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected exit panic")
		}
	}()
	_ = ParseArgsFrom([]string{"forst", "run", "-help"}, log)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d", exitCode)
	}
}

func TestParseArgsFrom_flagParseErrorReturnsEmpty(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	args := ParseArgsFrom([]string{"forst", "run", "-notaflag", "x.ft"}, log)
	if args.Command != "" {
		t.Fatalf("args = %+v", args)
	}
}

func TestParseArgsFrom_invalidRootAbs(t *testing.T) {
	orig := filepathAbsForArgs
	t.Cleanup(func() { filepathAbsForArgs = orig })
	filepathAbsForArgs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	args := ParseArgsFrom([]string{"forst", "run", "-root", "/bad", "x.ft"}, log)
	if args.Command != "" {
		t.Fatalf("args = %+v", args)
	}
	if !strings.Contains(buf.String(), "invalid -root") {
		t.Fatalf("log = %q", buf.String())
	}
}

func TestParseArgsFrom_shortHelpExits(t *testing.T) {
	var exitCode int
	orig := osExit
	t.Cleanup(func() { osExit = orig })
	osExit = func(code int) { exitCode = code; panic("exit") }

	log := logrus.New()
	log.SetOutput(io.Discard)
	defer func() { _ = recover() }()
	_ = ParseArgsFrom([]string{"forst", "-h"}, log)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d", exitCode)
	}
}
