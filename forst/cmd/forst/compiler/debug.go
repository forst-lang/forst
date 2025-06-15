package compiler

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	"runtime"
	"strings"

	logrus "github.com/sirupsen/logrus"
)

// logMemUsage logs the memory usage of the compiler
func (c *Compiler) logMemUsage(phase string, before, after runtime.MemStats) {
	if !c.Args.ReportMemoryUsage {
		return
	}

	allocDelta := after.TotalAlloc - before.TotalAlloc
	heapDelta := after.HeapAlloc - before.HeapAlloc

	c.log.WithFields(logrus.Fields{
		"phase":          phase,
		"allocatedBytes": allocDelta,
		"heapBytes":      heapDelta,
	}).Info("Memory usage")
}

func (c *Compiler) debugPrintTokens(tokens []ast.Token) {
	c.log.Debug("=== Tokens ===")
	for _, t := range tokens {
		c.log.WithFields(logrus.Fields{
			"location": fmt.Sprintf("%s:%d:%d", t.Path, t.Line, t.Column),
			"type":     string(t.Type),
			"value":    t.Value,
		}).Debug("Token")
	}
}

func (c *Compiler) debugPrintForstAST(forstAST []ast.Node) {
	c.log.Debug("=== Forst AST ===")
	for _, node := range forstAST {
		switch n := node.(type) {
		case ast.PackageNode:
			c.log.WithField("package", n.Ident).Debug("Package declaration")
		case ast.ImportNode:
			c.log.WithField("path", n.Path).Debug("Import")
		case ast.ImportGroupNode:
			c.log.WithField("importGroup", n.Imports).Debug("Import group")
		case ast.FunctionNode:
			fields := logrus.Fields{
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
			c.log.WithFields(fields).Debug("Function declaration")
		}
	}
}

func (c *Compiler) debugPrintGoAST(goFile *goast.File) {
	c.log.Debug("=== Go AST ===")
	c.log.WithField("package", goFile.Name).Debug("Package")

	c.log.Debug("Imports")
	for _, imp := range goFile.Imports {
		c.log.WithField("path", imp.Path.Value).Debug("Import")
	}

	c.log.Debug("Declarations")
	for _, decl := range goFile.Decls {
		switch d := decl.(type) {
		case *goast.FuncDecl:
			c.log.WithField("name", d.Name.Name).Debug("Function")
		case *goast.GenDecl:
			c.log.WithField("token", d.Tok).Debug("GenDecl")
		}
	}
}

func (c *Compiler) debugPrintTypeInfo(tc *typechecker.TypeChecker) {
	c.log.Debug("\n=== Type Check Results ===")

	c.log.Debug("Functions:")
	for id, sig := range tc.Functions {
		params := make([]string, len(sig.Parameters))
		for i, param := range sig.Parameters {
			params[i] = fmt.Sprintf("%s: %s", param.GetIdent(), param.Type)
		}

		returnTypes := make([]string, len(sig.ReturnTypes))
		for i, rt := range sig.ReturnTypes {
			returnTypes[i] = rt.String()
		}

		c.log.WithFields(logrus.Fields{
			"function":    id,
			"parameters":  params,
			"returnTypes": returnTypes,
		}).Debug("function signature")
	}

	c.log.Debug("Definitions:")
	for id, def := range tc.Defs {
		expr := ""
		if typeDef, ok := def.(ast.TypeDefNode); ok {
			expr = typeDef.Expr.String()
		}
		c.log.WithFields(logrus.Fields{
			"definition": id,
			"type":       fmt.Sprintf("%T", def),
			"expr":       expr,
		}).Debug("definition")
	}
}
