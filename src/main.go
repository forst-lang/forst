package main

import "fmt"

func main() {
	// Example Forst function with an assertion
	forstCode := `fn greet -> String { 
		assert 5 > 3 or ValidationError
		return "Hello, Forst!"
	}`

	// Compilation pipeline
	tokens := Lexer(forstCode)
	parser := NewParser(tokens)
	forstAST := parser.Parse()
	goAST := TransformForstToGo(forstAST)
	goCode := GenerateGoCode(goAST)

	// Output generated Go code
	fmt.Println(goCode)
}