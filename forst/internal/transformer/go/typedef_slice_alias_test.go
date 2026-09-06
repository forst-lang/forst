package transformergo

import (
	"strings"
	"testing"
)

func TestPipeline_sliceAliasTypedef_emitsValidGo(t *testing.T) {
	t.Parallel()
	src := `package main

type Bytes = []Byte
type Ints = []Int

func main() {
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	if strings.Contains(out, "TYPE_ARRAY") {
		t.Fatalf("emitted invalid TYPE_ARRAY:\n%s", out)
	}
	if !strings.Contains(out, "type Bytes []byte") {
		t.Fatalf("want type Bytes []byte, got:\n%s", out)
	}
	if !strings.Contains(out, "type Ints []int") {
		t.Fatalf("want type Ints []int, got:\n%s", out)
	}
	assertGoParses(t, out)
}

func TestParseTypeDef_preservesSliceTypeParams(t *testing.T) {
	t.Parallel()
	src := `package main
type Bytes = []Byte
`
	// Covered by pipeline test; keep a focused typecheck+emit smoke via compile.
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, "type Bytes []byte") {
		t.Fatalf("got:\n%s", out)
	}
}

func TestPipeline_fieldAfterCall_andIndexThenField(t *testing.T) {
	t.Parallel()
	src := `package main

type Q = {n: Int}
type Item = {name: String}

func makeQ(): Q {
	return Q{n: 7}
}

func main() {
	_ := makeQ().n
	xs := [Item{name: "a"}]
	_ := xs[0].name
}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, "makeQ().N") && !strings.Contains(out, "makeQ().n") {
		// Exported struct fields capitalize; accept either depending on export flag.
		t.Fatalf("expected field access on call result, got:\n%s", out)
	}
	assertGoParses(t, out)
}
