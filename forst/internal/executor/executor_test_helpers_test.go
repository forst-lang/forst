package executor

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forst/internal/compiler"

	"github.com/sirupsen/logrus"
)

// walkForstFilesConfig finds .ft files under rootDir (same behavior as production discovery needs).
type walkForstFilesConfig struct{}

func (walkForstFilesConfig) FindForstFiles(rootDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".ft") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func testExecutor(t *testing.T, root string) *FunctionExecutor {
	t.Helper()
	log := logrus.New()
	log.SetOutput(io.Discard)
	comp := compiler.New(compiler.Args{Command: "run", FilePath: filepath.Join(root, "placeholder.ft")}, log)
	return NewFunctionExecutor(root, comp, log, walkForstFilesConfig{})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
