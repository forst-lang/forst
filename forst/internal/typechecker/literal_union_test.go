package typechecker

import (
	"strings"
	"testing"

	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestLiteralUnion_inclusionAssignability(t *testing.T) {
	src := `package main

type TaskStatus = "todo" | "in_progress" | "success" | "failed"
type PendingStatus = "todo" | "in_progress"

func main() {
	var p: PendingStatus = "todo"
	var x: TaskStatus = p
	println(x)
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := New(log, false).CheckTypes(nodes); err != nil {
		t.Fatal(err)
	}
}

func TestLiteralUnion_rejectNonMemberLiteral(t *testing.T) {
	src := `package main

type PendingStatus = "todo" | "in_progress"

func main() {
	var x: PendingStatus = "failed"
	println(x)
}
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	err = New(log, false).CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected type error")
	}
	if !strings.Contains(err.Error(), "assignment type mismatch") {
		t.Fatalf("got %v", err)
	}
}

func TestLiteralUnion_rejectMixedKinds(t *testing.T) {
	src := `package main

type Bad = "todo" | 1
`
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)
	nodes, err := parser.NewTestParser(src, log).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	err = New(log, false).CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected type error")
	}
	if !strings.Contains(err.Error(), "refinement-unsupported-union") {
		t.Fatalf("got %v", err)
	}
}
