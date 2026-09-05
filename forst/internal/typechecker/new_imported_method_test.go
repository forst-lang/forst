package typechecker

import (
	"os"
	"path/filepath"
	"testing"

	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/testmod"

	"github.com/sirupsen/logrus"
)

// Tip handoff 05: method calls after new(imported.Named) need Go type binding.
func TestMethodCall_afterNewImportedNamedType(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(testmod.GoModContent("newprobe")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "store.go"), []byte(`package store

type Store struct{ N int }

func (s *Store) Get() int { return s.N }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	src := `package main

import (
	"fmt"
	"newprobe/store"
)

func main() {
	st := new(store.Store)
	fmt.Println(st.Get())
}
`
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)
	toks := lexer.New([]byte(src), "main.ft", log).Lex()
	nodes, err := parser.New(toks, "main.ft", log).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	tc.GoWorkspaceDir = root
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
}
