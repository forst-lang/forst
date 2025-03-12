package main

import (
	"fmt"
	"forst/pkg/generators"
	"forst/pkg/lexer"
	"forst/pkg/parser"
	"forst/pkg/transformers"
)

func main() {
	// Example Forst function with an assertion
	forstCode := `fn greet -> String { 
		assert true or ValidationError
		return "Hello, Forst!"
	}`

	// Compilation pipeline
	tokens := lexer.Lexer(forstCode)
	parser := parser.NewParser(tokens)
	forstAST := parser.Parse()
	goAST := transformers.TransformForstToGo(forstAST)
	goCode := generators.GenerateGoCode(goAST)

	// Output generated Go code
	fmt.Println(goCode)
}