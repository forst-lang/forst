package transformergo

import (
	"strings"
	"testing"
)

func TestPipeline_ensureMeetMinMax_andsSuccessPolarity(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	s := "hello"
	ensure s is Min(3).Max(10)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`utf8.RuneCountInString`,
		`< 3`,
		`> 10`,
		`||`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureHasPrefixOrHasPrefix(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	s := "+1"
	ensure s is HasPrefix("+") or HasPrefix("0")
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `strings.HasPrefix`) {
		t.Fatalf("expected HasPrefix\n----\n%s\n----", out)
	}
	// Failure of `A or B` is `!A && !B` (De Morgan), not `!(A || B)`.
	if !strings.Contains(out, `&&`) || !strings.Contains(out, `!strings.HasPrefix(s, "+")`) {
		t.Fatalf("expected De Morgan failure of OR alternatives\n----\n%s\n----", out)
	}
}

func TestPipeline_ensureHasSuffix(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	s := "x.md"
	ensure s is HasSuffix(".md")
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `strings.HasSuffix`) || !strings.Contains(out, `.md`) {
		t.Fatalf("expected HasSuffix\n----\n%s\n----", out)
	}
}

func TestPipeline_ensureMaxBytes(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	s := "ab"
	ensure s is MaxBytes(2)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `len(s)`) || !strings.Contains(out, `<= 2`) {
		t.Fatalf("expected MaxBytes len check\n----\n%s\n----", out)
	}
	if strings.Contains(out, `RuneCountInString`) {
		t.Fatalf("MaxBytes must not use RuneCount\n----\n%s\n----", out)
	}
}

func TestPipeline_ensureFloatFiniteAndLessThan(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	x := 0.5
	ensure x is Finite()
	ensure x is LessThan(1.0)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{`math.IsInf`, `math.IsNaN`, `"math"`, `x >= 1`} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureMapMin(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	m := map[String]Int{"a": 1}
	ensure m is Min(1)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `len(m)`) {
		t.Fatalf("expected map len bound\n----\n%s\n----", out)
	}
}
