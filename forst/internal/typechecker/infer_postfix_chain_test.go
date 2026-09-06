package typechecker

import (
	"testing"

	"forst/internal/testutil"
)

func TestCheckTypes_methodChain_onForstReceiver(t *testing.T) {
	t.Parallel()
	src := `package main

type Q = {n: Int}

func (q Q) Inc(): Q {
	return Q{n: 1}
}

func (q Q) Value(): Int {
	return q.n
}

func main() {
	q := Q{n: 0}
	_ := q.Inc().Value()
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_fieldAfterCall(t *testing.T) {
	t.Parallel()
	src := `package main

type Q = {n: Int}

func makeQ(): Q {
	return Q{n: 7}
}

func main() {
	_ := makeQ().n
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_indexThenField(t *testing.T) {
	t.Parallel()
	src := `package main

type Item = {name: String}

func main() {
	xs := [Item{name: "a"}]
	_ := xs[0].name
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}

func TestCheckTypes_methodOnCallResult(t *testing.T) {
	t.Parallel()
	src := `package main

type Q = {n: Int}

func makeQ(): Q {
	return Q{n: 0}
}

func (q Q) Inc(): Q {
	return Q{n: 1}
}

func main() {
	_ := makeQ().Inc()
}
`
	MustTypecheck(t, src, testutil.TypecheckOpts{})
}
