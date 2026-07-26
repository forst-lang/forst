package transformergo

import (
	"strings"
	"testing"
)

func TestTransformBitwise_emitsGoOperators(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	n := 1
	n = n ^ 2
	n = n << 3
	n = n >> 1
	n = n &^ 4
	n ^= 5
	n <<= 2
	n >>= 1
	n &^= 3
}
`
	out := compileForstPipeline(t, src)
	for _, want := range []string{
		"n ^ 2",
		"n << 3",
		"n >> 1",
		"n &^ 4",
		"n ^= 5",
		"n <<= 2",
		"n >>= 1",
		"n &^= 3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	assertGoParses(t, out)
}
