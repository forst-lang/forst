package main

import (
	"flag"
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/generators"
	"forst/pkg/lexer"
	"forst/pkg/parser"
	transformer_go "forst/pkg/transformer/go"
	"forst/pkg/typechecker"
	goast "go/ast"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type ProgramArgs struct {
	filePath string
	debug    bool
	trace    bool
}

func parseArgs() ProgramArgs {
	debug := flag.Bool("debug", false, "Enable debug output")
	trace := flag.Bool("trace", false, "Enable trace output")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: forst <filename>.ft")
		return ProgramArgs{}
	}
	return ProgramArgs{
		filePath: args[0],
		debug:    *debug,
		trace:    *trace,
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
	type tokenPrint struct {
		location string
		tokType  string
		value    string
	}

	prints := make([]tokenPrint, len(tokens))
	maxLocWidth := 0
	maxTypeWidth := 0
	for i, t := range tokens {
		loc := fmt.Sprintf("%s:%d:%d", t.Path, t.Line, t.Column)
		tokType := string(t.Type)
		if len(loc) > maxLocWidth {
			maxLocWidth = len(loc)
		}
		if len(tokType) > maxTypeWidth {
			maxTypeWidth = len(tokType)
		}
		prints[i] = tokenPrint{
			location: loc,
			tokType:  tokType,
			value:    t.Value,
		}
	}

	fmt.Println("\n=== Tokens ===")
	for _, p := range prints {
		fmt.Printf("%-*s   %-*s '%s'\n",
			maxLocWidth, p.location,
			maxTypeWidth, p.tokType,
			p.value)
	}
}

func debugPrintForstAST(forstAST []ast.Node) {
	fmt.Println("\n=== Forst AST ===")
	for _, node := range forstAST {
		switch n := node.(type) {
		case ast.PackageNode:
			fmt.Printf("  Package: %s\n", n.Ident)
			fmt.Println()
		case ast.ImportNode:
			fmt.Printf("  Import: %s\n", n.Path)
			fmt.Println()
		case ast.ImportGroupNode:
			fmt.Printf("  ImportGroup: %v\n", n.Imports)
			fmt.Println()
		case ast.FunctionNode:
			if n.HasExplicitReturnType() {
				returnTypes := make([]string, len(n.ReturnTypes))
				for i, rt := range n.ReturnTypes {
					returnTypes[i] = rt.String()
				}
				fmt.Printf("  Function: %s -> %s\n", n.Ident.Id, strings.Join(returnTypes, ", "))
			} else {
				fmt.Printf("  Function: %s -> (?)\n", n.Ident.Id)
			}
			for _, stmt := range n.Body {
				fmt.Printf("      %s\n", stmt)
			}
			fmt.Println()
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

func debugPrintTypeInfo(tc *typechecker.TypeChecker) {
	log.Debug("\n=== Type Check Results ===")

	log.Debug("  Functions:")
	for id, sig := range tc.Functions {
		log.Debugf("    %s(", id)
		for i, param := range sig.Parameters {
			if i > 0 {
				log.Debug(", ")
			}
			log.Debugf("%s: %s", param.Id(), param.Type)
		}
		returnTypes := make([]string, len(sig.ReturnTypes))
		for i, rt := range sig.ReturnTypes {
			returnTypes[i] = rt.String()
		}
		log.Debugf(") -> %s\n", strings.Join(returnTypes, ", "))
	}

	fmt.Println("  Definitions:")
	for id, def := range tc.Defs {
		fmt.Printf("  %s -> %T\n", id, def)
	}
}

func main() {
	args := parseArgs()
	if args.filePath == "" {
		return
	}

	if args.trace {
		log.SetLevel(log.TraceLevel)
	} else if args.debug {
		log.SetLevel(log.DebugLevel)
	}

	source, err := readSourceFile(args.filePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Info("Performing lexical analysis...")

	// Lexical Analysis
	tokens := lexer.Lexer(source, lexer.Context{FilePath: args.filePath})
	if args.debug {
		debugPrintTokens(tokens)
	}

	log.Info("Performing syntax analysis...")

	// Parsing
	forstNodes := parser.NewParser(tokens).ParseFile()
	if args.debug {
		debugPrintForstAST(forstNodes)
	}

	log.Info("Performing semantic analysis...")

	// Semantic Analysis
	checker := typechecker.New()

	// Collect, infer and check type
	if err := checker.CheckTypes(forstNodes); err != nil {
		log.Errorf("Type checking error: %v\n", err)
		log.Debug("  ")
		checker.DebugPrintCurrentScope()
		return
	}

	if args.debug {
		debugPrintTypeInfo(checker)
	}

	log.Info("Performing code generation...")

	// Transform to Go AST with type information
	transformer := transformer_go.New(checker)
	goAST, err := transformer.TransformForstFileToGo(forstNodes)
	if err != nil {
		log.Errorf("Transformation error: %v\n", err)
		return
	}

	if args.debug {
		debugPrintGoAST(goAST)
	}

	// Generate Go code
	goCode := generators.GenerateGoCode(goAST)

	if args.debug {
		log.Debug("\n=== Generated Go Code ===")
	}

	fmt.Println(goCode)
}
