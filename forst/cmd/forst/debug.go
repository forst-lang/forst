package main

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
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

	p.logger.WithFields(log.Fields{
		"phase":          phase,
		"allocatedBytes": allocDelta,
		"heapBytes":      heapDelta,
	}).Info("Memory usage")
}

func (p *Program) debugPrintTokens(tokens []ast.Token) {
	p.logger.Debug("=== Tokens ===")
	for _, t := range tokens {
		p.logger.WithFields(log.Fields{
			"location": fmt.Sprintf("%s:%d:%d", t.Path, t.Line, t.Column),
			"type":     string(t.Type),
			"value":    t.Value,
		}).Debug("Token")
	}
}

func (p *Program) debugPrintForstAST(forstAST []ast.Node) {
	p.logger.Debug("=== Forst AST ===")
	for _, node := range forstAST {
		switch n := node.(type) {
		case ast.PackageNode:
			p.logger.WithField("package", n.Ident).Debug("Package declaration")
		case ast.ImportNode:
			p.logger.WithField("path", n.Path).Debug("Import")
		case ast.ImportGroupNode:
			p.logger.WithField("importGroup", n.Imports).Debug("Import group")
		case ast.FunctionNode:
			fields := log.Fields{
				"name": n.GetIdent(),
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
			p.logger.WithFields(fields).Debug("Function declaration")
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
	p.logger.Debug("\n=== Type Check Results ===")

	p.logger.Debug("Functions:")
	for id, sig := range tc.Functions {
		params := make([]string, len(sig.Parameters))
		for i, param := range sig.Parameters {
			params[i] = fmt.Sprintf("%s: %s", param.GetIdent(), param.Type)
		}

		returnTypes := make([]string, len(sig.ReturnTypes))
		for i, rt := range sig.ReturnTypes {
			returnTypes[i] = rt.String()
		}

		p.logger.WithFields(log.Fields{
			"function":    id,
			"parameters":  params,
			"returnTypes": returnTypes,
		}).Debug("function signature")
	}

	p.logger.Debug("Definitions:")
	for id, def := range tc.Defs {
		expr := ""
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			expr = typeDef.Expr.String()
		}
		p.logger.WithFields(log.Fields{
			"definition": id,
			"type":       fmt.Sprintf("%T", def),
			"expr":       expr,
		}).Debug("definition")
	}
}
