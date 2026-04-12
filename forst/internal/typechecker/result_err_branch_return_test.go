package typechecker

import (
	"strings"
	"testing"

	"forst/internal/parser"
)

func TestResultErrBranch_returnErr_disallowed(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, String) {
	return 0
}

func g(): Result(Int, String) {
	x := f()
	if x is Err() {
		return Err("fail")
	}
	return 1
}

func main() {
	println(1)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Err is not a value constructor")
	}
	if !strings.Contains(err.Error(), "Err(...)") || !strings.Contains(err.Error(), "not a value constructor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResultErrBranch_returnOk_recoveryAllowed(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, String) {
	return 0
}

func g(): Result(Int, String) {
	x := f()
	if x is Err() {
		return 42
	}
	return 1
}

func main() {
	println(1)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestResultErrBranch_returnErr_outsideErrBranch_rejected(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func g(): Result(Int, String) {
	return Err("fail")
}

func main() {
	println(1)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Err is not a value constructor")
	}
	if !strings.Contains(err.Error(), "Err(...)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResultErrBranch_nestedTwoErrIfBranches_returnErrRejected(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

func f(): Result(Int, String) {
	return 0
}

func g(): Result(Int, String) {
	a := f()
	b := f()
	if a is Err() {
		if b is Err() {
			return Err("both")
		}
	}
	return 1
}

func main() {
	println(1)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Err is not a value constructor")
	}
	if !strings.Contains(err.Error(), "Err(...)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResultErrBranch_fieldPath_returnErr_rejected(t *testing.T) {
	t.Parallel()
	log := setupTestLogger(nil)
	src := `package main

type WrapStr = {
	r: Result(Int, String),
}

func fail(): Result(Int, String) {
	return 0
}

func g(): Result(Int, String) {
	x := fail()
	w := { r: x }
	if w.r is Err() {
		return Err("propagate")
	}
	return 1
}

func main() {
	println(1)
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: Err is not a value constructor")
	}
	if !strings.Contains(err.Error(), "Err(...)") {
		t.Fatalf("unexpected error: %v", err)
	}
}
