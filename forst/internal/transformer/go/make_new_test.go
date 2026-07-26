package transformergo

import (
	"strings"
	"testing"
)

func TestTransformMakeNew_emitsGoTypes(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	xs := make(Array(Int), 10, 20)
	m := make(map[String]Int, 4)
	p := new(Int)
	println(len(xs))
	println(len(m))
	if p != nil {
		println("ok")
	}
}
`
	out := compileForstPipeline(t, src)
	for _, want := range []string{
		"make([]int, 10, 20)",
		"make(map[string]int, 4)",
		"new(int)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	assertGoParses(t, out)
}
