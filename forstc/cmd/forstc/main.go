package main

import (
	"fmt"
	"forstc/pkg/generators"
	"forstc/pkg/lexer"
	"forstc/pkg/parser"
	"forstc/pkg/transformers"
)

func main() {
	// Example Forst function with an assertion
	forstCode := `fn greet -> String { 
		assert 5 > 3 or ValidationError
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