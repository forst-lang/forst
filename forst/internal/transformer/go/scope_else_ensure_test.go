package transformergo

import (
	"testing"

	"forst/internal/testutil"
)

func TestTransformIfNode_elseBranchVariableVisibleToBareEnsure(t *testing.T) {
	t.Parallel()
	src := `package demo

func main(): Void {
	flag := true
	if flag {
		println("a")
	} else {
		x := 5
		ensure x is GreaterThan(0)
		println("b")
	}
}
`
	MustCompileGo(t, src, testutil.CompileOpts{})
}

func TestTransformIfNode_elseIfBranchVariableVisibleToBareEnsure(t *testing.T) {
	t.Parallel()
	src := `package demo

func main(): Void {
	flag := 1
	if flag == 0 {
		println("a")
	} else if flag == 1 {
		x := 5
		ensure x is GreaterThan(0)
		println("b")
	}
}
`
	MustCompileGo(t, src, testutil.CompileOpts{})
}

func TestTransformIfNode_elseBranchResultVariableVisibleToEnsureOk(t *testing.T) {
	t.Parallel()
	src := `package demo

func okInt(): Result(Int, Error) {
	return 5
}

func main(): Void {
	flag := true
	if flag {
		println("a")
	} else {
		x := okInt()
		ensure x is Ok()
		println("b")
	}
}
`
	MustCompileGo(t, src, testutil.CompileOpts{})
}
