package main

import (
	"fmt"
	"forst/pkg/ast"
	"forst/pkg/typechecker"
	goast "go/ast"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (p *Program) logMemUsage(phase string, before, after runtime.MemStats) {
	if !p.Args.reportMemoryUsage {
		return
	}

	allocDelta := after.TotalAlloc - before.TotalAlloc
	heapDelta := after.HeapAlloc - before.HeapAlloc

	log.WithFields(log.Fields{
		"phase":          phase,
		"allocatedBytes": allocDelta,
		"heapBytes":      heapDelta,
	}).Info("Memory usage")
}

func (p *Program) debugPrintTokens(tokens []ast.Token) {
	log.Debug("=== Tokens ===")
	for _, t := range tokens {
		log.WithFields(log.Fields{
			"location": fmt.Sprintf("%s:%d:%d", t.Path, t.Line, t.Column),
			"type":     string(t.Type),
			"value":    t.Value,
		}).Debug("Token")
	}
}

func (p *Program) debugPrintForstAST(forstAST []ast.Node) {
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

func (p *Program) debugPrintGoAST(goFile *goast.File) {
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

func (p *Program) debugPrintTypeInfo(tc *typechecker.TypeChecker) {
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
