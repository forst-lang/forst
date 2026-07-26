package transformergo

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestTransformSwitch_emitsGoSwitchStmt(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	switch x := 1; x {
	case 1, 2:
		println("a")
		fallthrough
	default:
		println("b")
	}
	switch {
	case true:
		println("yes")
	}
}
`
	out := compileForstPipeline(t, src)
	for _, want := range []string{
		"switch x := 1; x {",
		"case 1, 2:",
		"fallthrough",
		"default:",
		"switch {",
		"case true:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	assertGoParses(t, out)
}

func TestTransformSwitchNode_shape(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	switch 2 {
	case 1:
		println("one")
	}
}
`
	out := compileForstPipeline(t, src)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", out, 0)
	if err != nil {
		t.Fatalf("parse emitted Go: %v", err)
	}
	var sw *goast.SwitchStmt
	goast.Inspect(file, func(n goast.Node) bool {
		if s, ok := n.(*goast.SwitchStmt); ok {
			sw = s
		}
		return true
	})
	if sw == nil {
		t.Fatal("expected SwitchStmt in output")
	}
	if sw.Tag == nil {
		t.Fatal("expected tag expression")
	}
	if sw.Body == nil || len(sw.Body.List) != 1 {
		t.Fatalf("case clauses: got %d want 1", len(sw.Body.List))
	}
	cc, ok := sw.Body.List[0].(*goast.CaseClause)
	if !ok {
		t.Fatalf("case clause type: %T", sw.Body.List[0])
	}
	if len(cc.List) != 1 {
		t.Fatalf("case values: got %d want 1", len(cc.List))
	}
}
