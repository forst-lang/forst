package transformergo

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestTransformGoto_emitsBranchAndLabeledStmt(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	goto done
	println(1)
done:
	println(2)
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{})
	if !strings.Contains(out, "goto done") {
		t.Fatalf("expected goto done in output, got:\n%s", out)
	}
	if !strings.Contains(out, "done:") {
		t.Fatalf("expected done: label in output, got:\n%s", out)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "out.go", out, 0); err != nil {
		t.Fatalf("generated Go does not parse: %v\n%s", err, out)
	}
}

func TestTransformGoto_backwardLoop(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	i := 0
retry:
	if i < 2 {
		i = i + 1
		goto retry
	}
	println(i)
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{})
	if !strings.Contains(out, "goto retry") || !strings.Contains(out, "retry:") {
		t.Fatalf("expected retry label/goto, got:\n%s", out)
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "out.go", out, 0); err != nil {
		t.Fatalf("generated Go does not parse: %v\n%s", err, out)
	}
}

func TestTransformGoto_labeledForBreak(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
outer:
	for {
		break outer
	}
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{})
	if !strings.Contains(out, "outer:") || !strings.Contains(out, "break outer") {
		t.Fatalf("expected labeled for/break, got:\n%s", out)
	}
}
