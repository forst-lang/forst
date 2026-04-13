package transformergo

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"
)

func TestPipeline_resultEnsure_emittedGoBuilds(t *testing.T) {
	t.Parallel()
	src := `package main

func positive(n Int): Result(Int, Error) {
	ensure n is GreaterThan(0)
	return n
}

func main() {
	println("ok")
}
`
	goCode := compileForstPipeline(t, src)
	assertGeneratedGoBuilds(t, goCode)
}

func TestPipeline_nominalErrorUnion_emittedGoBuilds(t *testing.T) {
	t.Parallel()
	src := `package main

error ParseError {
	message: String,
}

func parse(raw String): Result(String, Error) {
	if raw == "" {
		return "missing"
	}
	return raw
}

func main() {
	println("ok")
}
`
	goCode := compileForstPipeline(t, src)
	assertGeneratedGoBuilds(t, goCode)
}

func assertGeneratedGoBuilds(t *testing.T, goCode string) {
	t.Helper()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(goCode), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, mainPath, goCode, parser.AllErrors)
	if err != nil {
		t.Fatalf("go parse failed: %v\n\nGenerated:\n%s", err, goCode)
	}
	typeConfig := &types.Config{Importer: importer.Default()}
	if _, err := typeConfig.Check("generated", fileSet, []*ast.File{parsedFile}, nil); err != nil {
		t.Fatalf("go typecheck failed: %v\n\nGenerated:\n%s", err, goCode)
	}
}
