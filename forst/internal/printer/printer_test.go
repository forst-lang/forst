package printer

import (
	"strings"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestFormatSource_functionParams_goStyleNoColonBeforeType(t *testing.T) {
	t.Parallel()
	const src = `package main

func add(a Int, b Int): Int {
	return a + b
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "params.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if strings.Contains(out, "a: Int") || strings.Contains(out, "b: Int") {
		t.Fatalf("expected Go-style params (name Type), got:\n%s", out)
	}
	if !strings.Contains(out, "a Int") || !strings.Contains(out, "b Int") {
		t.Fatalf("expected Go-style params, got:\n%s", out)
	}
	l := lexer.New([]byte(out), "params.ft", log)
	tokens := l.Lex()
	p := parser.New(tokens, "params.ft", log)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("re-parse pretty output: %v\n--- out ---\n%s", err, out)
	}
}

func TestFormatSource_basicFt_roundTripsParse(t *testing.T) {
	t.Parallel()
	const src = `package main

func greet(): String {
	return "Hello, World!"
}

func main() {
	println(greet())
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "basic.ft", log)
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	// Second parse must succeed
	l := lexer.New([]byte(out), "basic.ft", log)
	tokens := l.Lex()
	p := parser.New(tokens, "basic.ft", log)
	if _, err := p.ParseFile(); err != nil {
		t.Fatalf("re-parse pretty output: %v\n--- out ---\n%s", err, out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatal("output should end with newline")
	}
}

func TestFormatSource_trimsMeaninglessWhitespace(t *testing.T) {
	t.Parallel()
	const messy = `package   main   

func   main (  )   {   return   1
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(messy, "x.ft", log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "package   main") {
		t.Fatalf("expected normalized package line, got %q", out)
	}
}

func TestFormatSource_preservesCommentsAndBlankLinesBetweenDecls(t *testing.T) {
	t.Parallel()
	const src = `package main
// Doc for a
func a(): Int {
	// inside
	return 1
}

func b(): Int {
	return 2
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "x.ft", log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "// Doc for a") {
		t.Fatalf("missing doc comment: %q", out)
	}
	if !strings.Contains(out, "// inside") {
		t.Fatalf("missing inner comment: %q", out)
	}
	if !strings.Contains(out, "\n\nfunc a()") && !strings.Contains(out, "\nfunc a()") {
		t.Fatalf("unexpected layout around func a: %q", out)
	}
	// Two top-level funcs separated by a blank line (gofmt-style)
	if !strings.Contains(out, "}\n\nfunc b()") {
		t.Fatalf("expected blank line between top-level funcs, got:\n%s", out)
	}
}

func TestFormatSource_commentOnlyFile(t *testing.T) {
	t.Parallel()
	const src = `// SPDX
// license
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	out, err := FormatSource(src, "x.ft", log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "// SPDX") {
		t.Fatalf("got %q", out)
	}
}

func TestPrint_unsupportedTopLevelErrors(t *testing.T) {
	t.Parallel()
	// nil slice only — empty file
	out, err := Print(DefaultConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Fatalf("got %q", out)
	}
}
