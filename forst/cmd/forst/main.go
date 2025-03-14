package main

import (
	"flag"
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/generators"
	"forst/pkg/lexer"
	"forst/pkg/parser"
	"forst/pkg/transformer"
	goast "go/ast"
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

func debugPrintForstAST(forstAST ast.FunctionNode) {
	fmt.Println("\n=== Forst AST ===")
	fmt.Printf("Function: %s -> %s\n", forstAST.Name, forstAST.ReturnType)
	fmt.Println("Body:")
	for _, node := range forstAST.Body {
		switch n := node.(type) {
		case ast.EnsureNode:
			fmt.Printf("  Ensure: %s or %s\n", n.Condition, n.ErrorType)
		case ast.ReturnNode:
			fmt.Printf("  Return: %s\n", n.Value)
		}
	}
}

func debugPrintGoAST(goAST *goast.FuncDecl) {
	fmt.Println("\n=== Go AST ===")
	fmt.Printf("Function: %s\n", goAST.Name.Name)
	
	// Print return type if exists
	if goAST.Type.Results != nil && len(goAST.Type.Results.List) > 0 {
		returnType := goAST.Type.Results.List[0].Type
		fmt.Printf("Return Type: %s\n", returnType)
	}
	
	fmt.Println("Body:")
	for _, stmt := range goAST.Body.List {
		fmt.Printf("  %T\n", stmt)
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

	goAST := transformer.TransformForstToGo(forstAST)
	if args.debug {
		debugPrintGoAST(goAST)
	}

	goCode := generators.GenerateGoCode(goAST)

	if args.debug {
		fmt.Println("\n=== Generated Go Code ===")
	}
	fmt.Println(goCode)
}