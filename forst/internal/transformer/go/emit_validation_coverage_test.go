package transformergo

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func assertGoParses(t *testing.T, out string) {
	t.Helper()
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "gen.go", out, parser.AllErrors); err != nil {
		t.Fatalf("generated Go does not parse: %v\n%s", err, out)
	}
}

func TestEmitValidation_mapLiteralAndIndex(t *testing.T) {
	src := `package main

func main() {
	m := map[String]Int{ "a": 1, "b": 2 }
	println(m["a"])
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `map[`) || !strings.Contains(out, `[`) {
		t.Fatalf("expected map literal emit:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_arrayLiteral(t *testing.T) {
	src := `package main

func main() {
	a := [1, 2, 3]
	println(a[0])
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `[]`) {
		t.Fatalf("expected slice literal:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_breakContinueInFor(t *testing.T) {
	src := `package main

func main() {
	for i := 0; i < 10; i = i + 1 {
		if i == 3 {
			continue
		}
		if i == 7 {
			break
		}
	}
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `break`) || !strings.Contains(out, `continue`) {
		t.Fatalf("expected break/continue:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_labeledBreak(t *testing.T) {
	src := `package main

func main() {
outer: for i := 0; i < 10; i = i + 1 {
		for j := 0; j < 10; j = j + 1 {
			if j == 5 {
				break outer
			}
		}
	}
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `outer:`) || !strings.Contains(out, `break outer`) {
		t.Fatalf("expected labeled break:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_pointerAndReference(t *testing.T) {
	src := `package main

func main() {
	x := 1
	p := &x
	y := *p
	println(y)
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `&x`) || !strings.Contains(out, `*p`) {
		t.Fatalf("expected pointer ops:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_ensurePresentAndNil(t *testing.T) {
	src := `package main

func main() {
	var p *Int = nil
	ensure p is Nil()
	x := 1
	px := &x
	ensure px is Present()
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `== nil`) {
		t.Fatalf("expected nil check emit:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_shapeLiteralWithBaseType(t *testing.T) {
	src := `package main

type Point = { x: Int, y: Int }

func main() {
	p: Point = { x: 1, y: 2 }
	println(p.x)
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`type Point`, `x:`, `y:`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestEmitValidation_deferStatement(t *testing.T) {
	src := `package main

func cleanup() {
	println("done")
}

func main() {
	defer cleanup()
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `defer cleanup()`) {
		t.Fatalf("expected defer emit:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_withBlockEmitsProvidersStruct(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {}

func main() {
	with {
		Logger: NopLogger{}
	} {
		println("ok")
	}
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `Logger`) || !strings.Contains(out, `NopLogger`) {
		t.Fatalf("expected with/providers emit:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_typedefUnionAndAlias(t *testing.T) {
	src := `package main

type ParseError = { message: String }
type IoError = { code: Int }
type AppError = ParseError | IoError

func main() {
	var e: AppError = ParseError{ message: "x" }
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`type ParseError`, `type IoError`, `message`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("missing %q in:\n%s", sub, out)
		}
	}
	assertGoParses(t, out)
}

func TestEmitValidation_useSiteInFunctionBody(t *testing.T) {
	src := `package main

type Logger = { info(msg String) }
type NopLogger = {}

func (NopLogger) info(msg String) {}

func work() {
	use logger: Logger
	logger.info("hi")
}

func main() {
	with { Logger: NopLogger{} } {
		work()
	}
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `Providers_`) || !strings.Contains(out, `work(providers`) {
		t.Fatalf("expected use/with lowering:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestEmitValidation_goImportAndDefer(t *testing.T) {
	src := `package main

import "os"

func main() {
	defer os.Exit(0)
}
`
	out := compileForstPipelineExt(t, src, pipelineOpts{goWorkspaceDir: moduleRootFromWD(t)})
	if !strings.Contains(out, `"os"`) || !strings.Contains(out, `defer os.Exit`) {
		t.Fatalf("expected os import emit:\n%s", out)
	}
	assertGoParses(t, out)
}
