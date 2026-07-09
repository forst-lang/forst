package transformergo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func examplesInRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "..", "examples", "in")
}

func TestPipeline_mergedProvidersDemo(t *testing.T) {
	t.Parallel()
	root := filepath.Join(examplesInRoot(t), "rfc", "providers")
	paths := []string{
		filepath.Join(root, "providers.ft"),
		filepath.Join(root, "providers_test.ft"),
		filepath.Join(root, "main_wiring.ft"),
	}
	out := compileMergedForstFilesPipeline(t, paths, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	for _, sub := range []string{
		`package providers_demo`,
		`func expireToken`,
		`func logExpiry`,
		`Providers_`,
		`import "testing"`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in providers merged output:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestPipeline_mergedImportsExample(t *testing.T) {
	t.Parallel()
	root := filepath.Join(examplesInRoot(t), "imports")
	paths := []string{
		filepath.Join(root, "main.ft"),
		filepath.Join(root, "greeting.ft"),
		filepath.Join(root, "cli.ft"),
		filepath.Join(root, "fmt_only.ft"),
	}
	importRoot := filepath.Join(root)
	out := compileMergedForstFilesPipeline(t, paths, pipelineOpts{goWorkspaceDir: importRoot})
	if !strings.Contains(out, `fmt.Println`) || !strings.Contains(out, `func greeting`) {
		t.Fatalf("unexpected imports merged output:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestPipeline_mergedTictactoePackage(t *testing.T) {
	t.Parallel()
	root := filepath.Join(examplesInRoot(t), "tictactoe")
	paths := []string{
		filepath.Join(root, "engine.ft"),
		filepath.Join(root, "server.ft"),
	}
	out := compileMergedForstFilesPipeline(t, paths, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	for _, sub := range []string{`func NewGame`, `func PlayMove`, `func main`, `fmt.Println`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in tictactoe merged output:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestPipeline_crossPkgProvidersAlphaLog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(examplesInRoot(t), "rfc", "providers", "cross_pkg", "auth", "log.ft")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := compileForstPipeline(t, string(src))
	if !strings.Contains(out, `func LogEvent`) || !strings.Contains(out, `Providers_`) {
		t.Fatalf("unexpected auth log output:\n%s", out)
	}
	assertGoParses(t, out)
}
