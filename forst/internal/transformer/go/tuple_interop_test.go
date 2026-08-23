package transformergo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitNominalErrorUnwrapMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		src        string
		wantUnwrap bool
		wantCause  bool
	}{
		{
			name:       "withCauseField",
			src:        "error WrapErr { cause: Error, msg: String }\nfunc main() {}",
			wantUnwrap: true,
			wantCause:  true,
		},
		{
			name:       "withoutCauseField",
			src:        "error PlainErr { msg: String }\nfunc main() {}",
			wantUnwrap: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			src := "package main\n" + tt.src
			out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
			hasUnwrap := strings.Contains(out, "func (e ")
			if tt.wantUnwrap {
				if !strings.Contains(out, "func (e WrapErr) Unwrap() error") {
					t.Fatalf("expected Unwrap method, got:\n%s", out)
				}
				if tt.wantCause && !strings.Contains(out, "return e.cause") {
					t.Fatalf("expected return e.cause, got:\n%s", out)
				}
			} else if hasUnwrap && strings.Contains(out, "Unwrap()") {
				t.Fatalf("did not expect Unwrap without cause field, got:\n%s", out)
			}
		})
	}
}

func TestTransformTupleAssignmentAndIndexAccess_emitsMultiValueLocals(t *testing.T) {
	t.Parallel()
	src := `package main
import "strconv"
func main() {
  x := strconv.Atoi("42")
  n := x.0
  println(n)
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "x0, _ := strconv.Atoi") {
		t.Fatalf("expected tuple split with blank for unused slot, got:\n%s", out)
	}
	if strings.Contains(out, "x0, x1 := strconv.Atoi") {
		t.Fatalf("did not expect unused x1 binding, got:\n%s", out)
	}
	if !strings.Contains(out, "x0") {
		t.Fatalf("expected x.0 to lower through x0 local, got:\n%s", out)
	}
	assertGoBuildsInTempModule(t, out)
}

func TestTransformTupleAssignment_partialUse_goBuilds(t *testing.T) {
	t.Parallel()
	dir := moduleRootFromWD(t)

	t.Run("partialAtoi", func(t *testing.T) {
		t.Parallel()
		src := `package main
import "strconv"

func demoAtoi(s: String) {
  pair := strconv.Atoi(s)
  n := pair.0
  println(n)
}

func main() {
  demoAtoi("42")
}
`
		out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: dir})
		if !strings.Contains(out, "pair0, _ := strconv.Atoi") {
			t.Fatalf("expected blank for unused Atoi error slot, got:\n%s", out)
		}
		assertGoBuildsInTempModule(t, out)
	})

	t.Run("fullCut", func(t *testing.T) {
		t.Parallel()
		src := `package main
import "strings"

func demoCut(s: String, sep: String) {
  t := strings.Cut(s, sep)
  before := t.0
  after := t.1
  found := t.2
  println(before)
  println(after)
  println(string(found))
}

func main() {
  demoCut("a,b", ",")
}
`
		out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: dir})
		if !strings.Contains(out, "t0, t1, t2 := strings.Cut") {
			t.Fatalf("expected all Cut slots named when all used, got:\n%s", out)
		}
		assertGoBuildsInTempModule(t, out)
	})
}

func TestTransformTupleAssignment_slotUsedInIfPredicate_goBuilds(t *testing.T) {
	t.Parallel()
	src := `package main
import "strings"

func main() {
  t := strings.Cut("a,b", ",")
  if t.2 {
    println(t.0)
  }
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "t0, _, t2 := strings.Cut") {
		t.Fatalf("expected t0 and t2 when predicate uses t.2 and body uses t.0, got:\n%s", out)
	}
	assertGoBuildsInTempModule(t, out)
}

func TestTransformTupleMultiValueReturn_emitsGoMultiReturn(t *testing.T) {
	t.Parallel()
	src := `package main
import "io"
func read(): Tuple(Int, Error) {
  return 0, io.EOF
}
func main() {
  println(0)
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "return 0, io.EOF") {
		t.Fatalf("expected multi-value return, got:\n%s", out)
	}
}

func TestGoExport_forstCanImplementIoReader(t *testing.T) {
	t.Parallel()
	src := `package main
import "io"

type ByteReader = { off: Int }

func (r ByteReader) Read(p: []byte): Tuple(Int, Error) {
  return 0, io.EOF
}

func main() {}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "func (r ByteReader) Read(p []byte) (int, error)") {
		t.Fatalf("expected io.Reader-shaped Read signature, got:\n%s", out)
	}
	harness := out + `

func assertByteReaderIsIOReader() {
	var _ io.Reader = ByteReader{}
}
`
	assertGoBuildsInTempModule(t, harness)
}

func assertGoBuildsInTempModule(t *testing.T, mainGo string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module forsttest\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "testbin"), ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}
