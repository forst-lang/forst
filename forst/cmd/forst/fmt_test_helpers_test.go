package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func testFmtLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func writeFmtTestFile(t *testing.T, dir, name, content string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func runFmt(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := runFmtCommand(args, testFmtLogger(), &out)
	return out.String(), err
}
