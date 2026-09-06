package typechecker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"forst/internal/goload"
	"forst/internal/testutil"
)

type goImportCorpus struct {
	Cases []struct {
		File          string `json:"file"`
		SamePackageGo string `json:"samePackageGo"`
		RequireImport string `json:"requireImport"`
	} `json:"cases"`
}

func TestGoImportCorpus_manifest(t *testing.T) {
	t.Parallel()
	path := testutil.ExamplePath(t, "go_interop/import-corpus.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus goImportCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("empty corpus")
	}
	for _, c := range corpus.Cases {
		c := c
		t.Run(c.File, func(t *testing.T) {
			t.Parallel()
			ftPath := testutil.ExamplePath(t, c.File)
			src, err := os.ReadFile(ftPath)
			if err != nil {
				t.Fatal(err)
			}
			opts := testutil.TypecheckOpts{FileID: filepath.Base(c.File)}
			dir := filepath.Dir(ftPath)
			opts.GoWorkspaceDir = goload.FindModuleRoot(dir)
			if c.SamePackageGo != "" {
				opts.SamePackageGoImport = c.SamePackageGo
			}
			if c.RequireImport != "" {
				opts.RequireGoImport = c.RequireImport
			}
			MustTypecheck(t, string(src), opts)
		})
	}
}
