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
func main() {
	// Parse command line flags
	debug := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: forst <filename>.ft")
		return
	}

	// Read the Forst file
	filePath := args[0]
	source, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	// Compilation pipeline
	tokens := lexer.Lexer(source, lexer.Context{FilePath: filePath})
	
	if *debug {
		fmt.Println("\n=== Tokens ===")
		for _, t := range tokens {
			fmt.Printf("%s:%d:%d: %-12s '%s'\n",
				t.Path, t.Line, t.Column,
				t.Type, t.Value)
		}
	}

	parser := parser.NewParser(tokens)
	forstAST := parser.Parse()

	if *debug {
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

	goAST := transformers.TransformForstToGo(forstAST)

	if *debug {
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

	goCode := generators.GenerateGoCode(goAST)

	// Output generated Go code
	if *debug {
		fmt.Println("\n=== Generated Go Code ===")
	}
	fmt.Println(goCode)
}