package transformergo

import (
	"strings"
	"testing"
)

// Covers transformEnsureCondition when the assertion is only a type-guard name (no Strong() call):
// BaseType set, Constraints empty — see ensure.go "type guard call" branch.
func TestPipeline_ensureTypeGuardNameOnly_emitsHashGuardCall(t *testing.T) {
	t.Parallel()
	src := `package main

type Password = String

is (password Password) Strong {
	ensure password is Min(12)
}

func main() {
	password: Password = "123456789012345"
	ensure password is Strong {
		println("weak")
	}
	println("done")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`func G_`,
		`!G_`,
		`(password)`,
		`os.Exit`,
		`package main`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

// "or NotOk(...)" is still lowered via an error return + errors.New today (nominal constructor not in failure path yet).
func TestPipeline_ensureOrNominalPayload_emitsErrorReturningFunc(t *testing.T) {
	t.Parallel()
	src := `package main

error NotOk {
	msg: String
}

func check() {
	n := 0
	ensure n is GreaterThan(0) or NotOk({ msg: "bad" })
}

func main() {
	check()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`type NotOk struct`,
		`func check() error`,
		`n <= 0`,
		`return errors.New("assertion failed: Int.GreaterThan(0)")`,
		`package main`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

// Exercises transformEnsureConstraint alias-chain retry: typedef does not map directly to builtinConstraints keys.
func TestPipeline_ensureBuiltinOnStringAlias_resolvesAliasChain(t *testing.T) {
	t.Parallel()
	src := `package main

type Label = String

func main() {
	s: Label = "hi"
	ensure s is Min(1)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`type Label string`,
		`len(s) < 1`,
		`os.Exit`,
		`func main`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureBuiltinOnIntAlias_resolvesAliasChain(t *testing.T) {
	t.Parallel()
	src := `package main

type Count = Int

func main() {
	n: Count = 5
	ensure n is GreaterThan(0)
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`type Count int`,
		`n <= 0`,
		`os.Exit`,
		`func main`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensurePointerNil_main_emitsNilCheck(t *testing.T) {
	t.Parallel()
	src := `package main

func main() {
	var p: *Int = nil
	ensure p is Nil()
	println("ok")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`var p *int = nil`,
		`p != nil`,
		`os.Exit`,
		`func main`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}
