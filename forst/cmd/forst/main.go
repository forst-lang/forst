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
	"runtime"
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
	log.Debug("=== Tokens ===")
	for _, t := range tokens {
		log.WithFields(log.Fields{
			"location": fmt.Sprintf("%s:%d:%d", t.Path, t.Line, t.Column),
			"type":     string(t.Type),
			"value":    t.Value,
		}).Debug("Token")
	}
}
func debugPrintForstAST(forstAST []ast.Node) {
	log.Debug("=== Forst AST ===")
	for _, node := range forstAST {
		switch n := node.(type) {
		case ast.PackageNode:
			log.WithField("package", n.Ident).Debug("Package declaration")
		case ast.ImportNode:
			log.WithField("path", n.Path).Debug("Import")
		case ast.ImportGroupNode:
			log.WithField("importGroup", n.Imports).Debug("Import group")
		case ast.FunctionNode:
			fields := log.Fields{
				"name": n.Ident.Id,
				"body": n.Body,
			}
			if n.HasExplicitReturnType() {
				returnTypes := make([]string, len(n.ReturnTypes))
				for i, rt := range n.ReturnTypes {
					returnTypes[i] = rt.String()
				}
				fields["returnTypes"] = strings.Join(returnTypes, ", ")
			} else {
				fields["returnTypes"] = "(?)"
			}
			log.WithFields(fields).Debug("Function declaration")
		}
	}
}

func debugPrintGoAST(goFile *goast.File) {
	log.Debug("=== Go AST ===")
	log.WithField("package", goFile.Name).Debug("Package")

	log.Debug("Imports")
	for _, imp := range goFile.Imports {
		log.WithField("path", imp.Path.Value).Debug("Import")
	}

	log.Debug("Declarations")
	for _, decl := range goFile.Decls {
		switch d := decl.(type) {
		case *goast.FuncDecl:
			log.WithField("name", d.Name.Name).Debug("Function")
		case *goast.GenDecl:
			log.WithField("token", d.Tok).Debug("GenDecl")
		}
	}
}

func debugPrintTypeInfo(tc *typechecker.TypeChecker) {
	log.Debug("\n=== Type Check Results ===")

	log.Debug("Functions:")
	for id, sig := range tc.Functions {
		params := make([]string, len(sig.Parameters))
		for i, param := range sig.Parameters {
			params[i] = fmt.Sprintf("%s: %s", param.Id(), param.Type)
		}

		returnTypes := make([]string, len(sig.ReturnTypes))
		for i, rt := range sig.ReturnTypes {
			returnTypes[i] = rt.String()
		}

		log.WithFields(log.Fields{
			"function":    id,
			"parameters":  params,
			"returnTypes": returnTypes,
		}).Debug("function signature")
	}

	log.Debug("Definitions:")
	for id, def := range tc.Defs {
		log.WithFields(log.Fields{
			"definition": id,
			"type":       fmt.Sprintf("%T", def),
		}).Debug("definition")
	}
}

func getMemStats() runtime.MemStats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return mem
}

func logMemUsage(phase string, before, after runtime.MemStats) {
	allocDelta := after.TotalAlloc - before.TotalAlloc
	heapDelta := after.HeapAlloc - before.HeapAlloc

	log.WithFields(log.Fields{
		"phase":          phase,
		"allocatedBytes": allocDelta,
		"heapBytes":      heapDelta,
	}).Info("Memory usage")
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

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableQuote:     true,
	})

	source, err := readSourceFile(args.filePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Info("Performing lexical analysis...")
	memBefore := getMemStats()

	// Lexical Analysis
	tokens := lexer.Lexer(source, lexer.Context{FilePath: args.filePath})

	memAfter := getMemStats()
	logMemUsage("lexical analysis", memBefore, memAfter)

	debugPrintTokens(tokens)

	log.Info("Performing syntax analysis...")
	memBefore = getMemStats()

	// Parsing
	forstNodes, err := parser.NewParser(tokens, args.filePath).ParseFile()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	memAfter = getMemStats()
	logMemUsage("syntax analysis", memBefore, memAfter)

	debugPrintForstAST(forstNodes)

	log.Info("Performing semantic analysis...")
	memBefore = getMemStats()

	// Semantic Analysis
	checker := typechecker.New()

	// Collect, infer and check type
	if err := checker.CheckTypes(forstNodes); err != nil {
		log.Errorf("Type checking error: %v\n", err)
		checker.DebugPrintCurrentScope()
		os.Exit(1)
	}

	memAfter = getMemStats()
	logMemUsage("semantic analysis", memBefore, memAfter)

	debugPrintTypeInfo(checker)

	log.Info("Performing code generation...")
	memBefore = getMemStats()

	// Transform to Go AST with type information
	transformer := transformer_go.New(checker)
	goAST, err := transformer.TransformForstFileToGo(forstNodes)
	if err != nil {
		log.Errorf("Transformation error: %v\n", err)
		os.Exit(1)
	}

	// Generate Go code
	goCode := generators.GenerateGoCode(goAST)

	debugPrintGoAST(goAST)

	memAfter = getMemStats()
	logMemUsage("code generation", memBefore, memAfter)

	log.Debug("=== Generated Go Code ===")

	fmt.Println(goCode)
}
