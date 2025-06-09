package transformergo

import (
	"forst/internal/ast"
	goast "go/ast"

	"github.com/sirupsen/logrus"
)

// transformTypeGuardEnsure transforms a type guard ensure
func (t *Transformer) transformTypeGuardEnsure(ensure ast.EnsureNode) goast.Expr {
	// Look up the real type guard node by name
	guardName := ensure.Assertion.Constraints[0].Name
	typeGuardNode := t.lookupTypeGuardNode(guardName)

	// Use hash-based guard function name
	hash := t.TypeChecker.Hasher.HashNode(*typeGuardNode)
	guardFuncName := hash.ToGuardIdent()

	if len(typeGuardNode.Parameters()) > 0 {
		switch typeGuardNode.Parameters()[0].(type) {
		case ast.SimpleParamNode:
			return &goast.CallExpr{
				Fun: goast.NewIdent(string(guardFuncName)),
				Args: []goast.Expr{
					t.transformExpression(ensure.Variable),
				},
			}
		case ast.DestructuredParamNode:
			panic("DestructuredParamNode not supported in type guard assertion")
		}
	} else {
		panic("Type guard has no parameters")
	}

	return &goast.CallExpr{
		Fun: goast.NewIdent(string(guardFuncName)),
		Args: []goast.Expr{
			t.transformExpression(ensure.Variable),
		},
	}
}

// Helper to look up a TypeGuardNode by name
func (t *Transformer) lookupTypeGuardNode(name string) *ast.TypeGuardNode {
	for _, def := range t.TypeChecker.Defs {
		if tg, ok := def.(ast.TypeGuardNode); ok {
			t.log.WithFields(logrus.Fields{
				"requested": name,
				"candidate": tg.GetIdent(),
			}).Trace("lookupTypeGuardNode: candidate check")
			if tg.GetIdent() == name {
				t.log.WithFields(logrus.Fields{
					"requested": name,
					"found":     true,
				}).Trace("lookupTypeGuardNode: found match")
				return &tg
			}
		}
	}
	t.log.WithFields(logrus.Fields{
		"requested": name,
		"found":     false,
	}).Trace("lookupTypeGuardNode: not found")
	panic("Type guard not found: " + name)
}

func (t *AssertionTransformer) isTypeGuardCompatible(varType ast.TypeNode, typeGuard *ast.TypeGuardNode) bool {
	t.transformer.log.WithFields(logrus.Fields{
		"varType":   varType.Ident,
		"typeGuard": typeGuard.GetIdent(),
	}).Trace("[isTypeGuardCompatible] Checking type guard compatibility")

	// Get the base type of the variable
	baseType, err := t.transformer.getEnsureBaseType(ast.EnsureNode{Variable: ast.VariableNode{ExplicitType: varType}})
	if err != nil {
		t.transformer.log.WithError(err).Error("[isTypeGuardCompatible] Failed to get base type")
		return false
	}
	t.transformer.log.WithFields(logrus.Fields{
		"baseType": baseType.Ident,
	}).Trace("[isTypeGuardCompatible] Base type")

	// Check if the type guard is defined for the base type
	for _, param := range typeGuard.Parameters() {
		t.transformer.log.WithFields(logrus.Fields{
			"paramType": param.GetType().Ident,
			"baseType":  baseType.Ident,
		}).Trace("[isTypeGuardCompatible] Checking parameter type")

		if param.GetType().Ident == baseType.Ident {
			t.transformer.log.WithFields(logrus.Fields{
				"typeGuard": typeGuard.GetIdent(),
				"baseType":  baseType.Ident,
			}).Trace("[isTypeGuardCompatible] Found compatible type guard")
			return true
		}
	}

	t.transformer.log.WithFields(logrus.Fields{
		"typeGuard": typeGuard.GetIdent(),
		"baseType":  baseType.Ident,
	}).Trace("[isTypeGuardCompatible] No compatible type guard found")
	return false
}
