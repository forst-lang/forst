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

// ensure-or with a nominal error payload lowers to returning the composite literal.
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
		`return NotOk{msg: "bad"}`,
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
		`len(string(s)) < 1`,
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

func TestPipeline_nominalErrorUnion_typedef_emitsMembersAndUnion(t *testing.T) {
	t.Parallel()
	src := `package main

error ParseError {
	code: Int,
}

error IoError {
	path: String,
}

type ErrKind = ParseError | IoError

func main() {
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`ParseError`,
		`IoError`,
		`ErrKind`,
		`package main`,
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

func TestPipeline_ensureFieldAccessInTest_emitsTestingFatalf(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

type Sample = { active: Bool, enabled: Bool }

func makeSample(): Sample {
  return Sample { active: false, enabled: true }
}

func TestSampleFields(t *testing.T) {
  s := makeSample()
  ensure s.active is False()
  ensure s.enabled is True()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`func TestSampleFields(t *testing.T) {`,
		`t.Helper()`,
		`t.Fatalf(`,
		`got %t`,
		`"s.active"`,
		`"Bool.False()"`,
		`"false"`,
		`"s.enabled"`,
		`"Bool.True()"`,
		`"true"`,
		`s.active`,
		`s.enabled`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
	for _, sub := range []string{
		`) error {`,
		`return errors.New`,
	} {
		if strings.Contains(out, sub) {
			t.Fatalf("generated Go should not contain %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureBoolInTest_emitsTestingFatalf(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func TestEnsureBool(t *testing.T) {
	ok := true
	ensure ok is True()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Helper()`,
		`t.Fatalf(`,
		`got %t`,
		`"ok"`,
		`"Bool.True()"`,
		`"true"`,
		`ok`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
	if strings.Contains(out, `panic("assertion failed")`) {
		t.Fatalf("expected no panic in test ensure, got:\n%s", out)
	}
}

func TestPipeline_ensureInNonTestFunction_stillPanicsOrReturnsError(t *testing.T) {
	t.Parallel()
	src := `package main

func check() {
	ok := true
	ensure ok is True()
}

func main() {}
`
	out := compileForstPipeline(t, src)
	if !strings.Contains(out, `func check() error`) {
		t.Fatalf("expected error return for ensure in non-test function, got:\n%s", out)
	}
	if !strings.Contains(out, `ensure ok is Bool.True(): want true`) {
		t.Fatalf("expected ensure want hint in non-test function, got:\n%s", out)
	}
}

func TestPipeline_ensureStringMinInTest_emitsLenGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func TestEnsureStringMin(t *testing.T) {
	s := ""
	ensure s is Min(1)
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got len=%d (%q)`,
		`len(`,
		`s`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureResultOkInTest_emitsErrGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func f(): Result(Int, Error) {
	return 1
}

func TestEnsureResultOk(t *testing.T) {
	x := f()
	ensure x is Ok()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got err=%v`,
		`xErr`,
		`"Ok()"`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureStringHasPrefixInTest_emitsQuotedGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func TestEnsureStringHasPrefix(t *testing.T) {
	s := "http://"
	ensure s is HasPrefix("https://")
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got %q`,
		`s`,
		`"prefix \"https://\""`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureStringAliasMinInTest_emitsStringCoercionInLenGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

type Label = String

func TestEnsureStringAliasMin(t *testing.T) {
	s: Label = ""
	ensure s is Min(1)
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got len=%d (%q)`,
		`len(string(s))`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureResultErrInTest_emitsErrGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func f(): Result(Int, Error) {
	return 1
}

func TestEnsureResultErr(t *testing.T) {
	x := f()
	ensure x is Err()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got err=%v`,
		`xErr`,
		`"Err()"`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureResultOkWithPayloadInTest_emitsErrAndValGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func f(): Result(Int, Error) {
	return 1
}

func TestEnsureResultOkPayload(t *testing.T) {
	x := f()
	ensure x is Ok(42)
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got err=%v, val=%v`,
		`xErr`,
		`x`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensurePointerNilInTest_emitsDefaultGot(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func TestEnsurePointerNil(t *testing.T) {
	var p: *Int = nil
	ensure p is Nil()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`t.Fatalf(`,
		`got %v, want %s`,
		`p`,
		`"nil"`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
}

func TestPipeline_ensureInTest_usesCustomTestingTParamName(t *testing.T) {
	t.Parallel()
	src := `package demo

import "testing"

func TestEnsureCustomHarness(tt *testing.T) {
	ok := true
	ensure ok is True()
}
`
	out := compileForstPipeline(t, src)
	for _, sub := range []string{
		`tt.Helper()`,
		`tt.Fatalf(`,
		`got %t`,
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("generated Go missing %q\n----\n%s\n----", sub, out)
		}
	}
	if strings.Contains(out, `func TestEnsureCustomHarness(t *testing.T)`) {
		t.Fatalf("expected custom harness param tt, not t:\n%s", out)
	}
}
