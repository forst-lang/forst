package transformergo

import (
	"strings"
	"testing"
)

func TestPipeline_sliceAliasTypedef_emitsValidGo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "Bytes",
			src: `package main

type Bytes = []Byte

func main() {
	println("ok")
}
`,
			want: "type Bytes []byte",
		},
		{
			name: "Ints",
			src: `package main

type Ints = []Int

func main() {
	println("ok")
}
`,
			want: "type Ints []int",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := compileForstPipeline(t, tt.src)
			if strings.Contains(out, "TYPE_ARRAY") {
				t.Fatalf("emitted invalid TYPE_ARRAY:\n%s", out)
			}
			if !strings.Contains(out, tt.want) {
				t.Fatalf("want %q, got:\n%s", tt.want, out)
			}
			assertGoParses(t, out)
		})
	}
}

func TestPipeline_bytesAliasTypedef_emitsValidGo(t *testing.T) {
	t.Parallel()
	src := `package main
type Bytes = []Byte
`
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
