package transformergo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitNominalErrorUnwrapMethod_withCauseField(t *testing.T) {
	t.Parallel()
	src := `package main
error WrapErr { cause: Error, msg: String }
func main() {}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, "func (e WrapErr) Unwrap() error") {
		t.Fatalf("expected Unwrap method, got:\n%s", out)
	}
	if !strings.Contains(out, "return e.cause") {
		t.Fatalf("expected return e.cause, got:\n%s", out)
	}
}

func TestEmitNominalErrorUnwrapMethod_withoutCauseField(t *testing.T) {
	t.Parallel()
	src := `package main
error PlainErr { msg: String }
func main() {}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if strings.Contains(out, "func (e PlainErr) Unwrap()") {
		t.Fatalf("did not expect Unwrap without cause field, got:\n%s", out)
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
	if !strings.Contains(out, "x0, x1 := strconv.Atoi") {
		t.Fatalf("expected tuple split assignment, got:\n%s", out)
	}
	if !strings.Contains(out, "x0") {
		t.Fatalf("expected x.0 to lower through x0 local, got:\n%s", out)
	}
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
