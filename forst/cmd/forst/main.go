package main

import (
	"flag"
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/generators"
	"forst/pkg/lexer"
	"forst/pkg/parser"
	transformer_go "forst/pkg/transformer/go"
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

func debugPrintForstAST(forstAST []ast.Node) {
	fmt.Println("\n=== Forst AST ===")
	for _, node := range forstAST {
		switch n := node.(type) {
		case ast.PackageNode:
			fmt.Printf("  Package: %s\n", n.Value)
		case ast.ImportNode:
			fmt.Printf("  Import: %s\n", n.Path)
		case ast.ImportGroupNode:
			fmt.Printf("  ImportGroup: %v\n", n.Imports)
		case ast.FunctionNode:
			if n.ExplicitReturnType.IsImplicit() {
				fmt.Printf("  Function: %s -> %s (implicit)\n", n.Name, n.ImplicitReturnType)
			} else {
				fmt.Printf("  Function: %s -> %s (explicit)\n", n.Name, n.ExplicitReturnType)
			}
		case ast.EnsureNode:
			if n.ErrorType != nil {
				fmt.Printf("  Ensure: %s or %s\n", n.Assertion, *n.ErrorType)
			} else {
				fmt.Printf("  Ensure: %s\n", n.Assertion)
			}
		case ast.ReturnNode:
			fmt.Printf("  Return: %s\n", n.Value)
		}
	}
}

func debugPrintGoAST(goFile *goast.File) {
	fmt.Println("\n=== Go AST ===")
	fmt.Printf("  Package: %s\n", goFile.Name)

	fmt.Println("  Imports:")
	for _, imp := range goFile.Imports {
		fmt.Printf("    %s\n", imp.Path.Value)
	}

	fmt.Println("  Declarations:")
	for _, decl := range goFile.Decls {
		switch d := decl.(type) {
		case *goast.FuncDecl:
			fmt.Printf("    Function: %s\n", d.Name.Name)
		case *goast.GenDecl:
			fmt.Printf("    GenDecl: %s\n", d.Tok)
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

	forstNodes := parser.NewParser(tokens).ParseFile()
	if args.debug {
		debugPrintForstAST(forstNodes)
	}

	goAST := transformer_go.TransformForstFileToGo(forstNodes)
	if args.debug {
		debugPrintGoAST(goAST)
	}

	goCode := generators.GenerateGoCode(goAST)

	if args.debug {
		fmt.Println("\n=== Generated Go Code ===")
	}
	fmt.Println(goCode)
}
