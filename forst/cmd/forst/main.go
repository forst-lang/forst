package main

import (
	"flag"
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/generators"
	"forst/pkg/lexer"
	"forst/pkg/parser"
	"forst/pkg/transformers"
	"os"
)

type ProgramArgs struct {
	filePath string
	debug    bool
}

func parseArgs() ProgramArgs {
	debug := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: forst <filename>.ft")
		return ProgramArgs{}
	}
	return ProgramArgs{
		filePath: args[0],
		debug:    *debug,
	}
}

func readSourceFile(filePath string) ([]byte, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	return source, nil
}

func debugPrintTokens(tokens []ast.Token) {
	fmt.Println("\n=== Tokens ===")
	for _, t := range tokens {
		fmt.Printf("%s:%d:%d: %-12s '%s'\n",
			t.Path, t.Line, t.Column,
			t.Type, t.Value)
	}
}

func debugPrintForstAST(forstAST ast.FuncNode) {
	fmt.Println("\n=== Forst AST ===")
	fmt.Printf("Function: %s -> %s\n", forstAST.Name, forstAST.ReturnType)
	fmt.Println("Body:")
	for _, node := range forstAST.Body {
		switch n := node.(type) {
		case ast.AssertNode:
			fmt.Printf("  Assert: %s or %s\n", n.Condition, n.ErrorType)
		case ast.ReturnNode:
			fmt.Printf("  Return: %s\n", n.Value)
		}
	}
}

func debugPrintGoAST(goAST ast.FuncNode) {
	fmt.Println("\n=== Go AST ===")
	fmt.Printf("Function: %s -> %s\n", goAST.Name, goAST.ReturnType)
	fmt.Println("Body:")
	for _, node := range goAST.Body {
		switch n := node.(type) {
		case ast.AssertNode:
			fmt.Printf("  Assert: %s\n", n.Condition)
		case ast.ReturnNode:
			fmt.Printf("  Return: %s\n", n.Value)
		}
	}
}

func main() {
	args := parseArgs()
	if args.filePath == "" {
		return
	}

	source, err := readSourceFile(args.filePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Compilation pipeline
	tokens := lexer.Lexer(source, lexer.Context{FilePath: args.filePath})
	if args.debug {
		debugPrintTokens(tokens)
	}

	forstAST := parser.NewParser(tokens).Parse()
	if args.debug {
		debugPrintForstAST(forstAST)
	}

	goAST := transformers.TransformForstToGo(forstAST)
	if args.debug {
		debugPrintGoAST(goAST)
	}

	goCode := generators.GenerateGoCode(goAST)

	if args.debug {
		fmt.Println("\n=== Generated Go Code ===")
	}
	fmt.Println(goCode)
}