package main

import (
	"flag"
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/generators"
	"forst/pkg/lexer"
	"forst/pkg/parser"
	"forst/pkg/transformers"
)

func main() {
	// Parse command line flags
	debug := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	// Example Forst function with an assertion
	forstCode := `fn greet -> String { 
		assert true or ValidationError
		return "Hello, Forst!"
	}`

	// Compilation pipeline
	tokens := lexer.Lexer(forstCode)
	
	if *debug {
		fmt.Println("\n=== Tokens ===")
		for _, t := range tokens {
			fmt.Printf("%s[%s] at line %d, col %d: '%s'\n",
				t.Type, t.Filename, t.Line, t.Column, t.Value)
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