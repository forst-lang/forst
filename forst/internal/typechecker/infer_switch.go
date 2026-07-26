package typechecker

import (
	"fmt"

	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferSwitchStatement(n *ast.SwitchNode) ([]ast.TypeNode, error) {
	if n == nil {
		return nil, nil
	}
	tc.log.WithFields(logrus.Fields{
		"function": "inferSwitchStatement",
		"hasTag":   n.Tag != nil,
	}).Trace("Type-checking switch statement")

	tc.pushScope(n)
	defer tc.popScope()

	if n.Init != nil {
		if _, err := tc.inferNodeType(n.Init); err != nil {
			return nil, err
		}
	}

	var tagTypes []ast.TypeNode
	if n.Tag != nil {
		var err error
		tagTypes, err = tc.inferExpressionType(n.Tag)
		if err != nil {
			return nil, err
		}
	}

	tc.switchDepth++
	defer func() { tc.switchDepth-- }()

	seenCaseKeys := make(map[string]struct{})
	for _, clause := range n.Clauses {
		if err := tc.inferSwitchClause(n.Tag, tagTypes, clause, seenCaseKeys); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (tc *TypeChecker) inferSwitchClause(
	tag ast.ExpressionNode,
	tagTypes []ast.TypeNode,
	clause ast.SwitchClauseNode,
	seenCaseKeys map[string]struct{},
) error {
	if clause.IsDefault {
		return tc.inferSwitchClauseBody(clause.Body)
	}
	for _, val := range clause.Values {
		valTypes, err := tc.inferExpressionType(val)
		if err != nil {
			return err
		}
		if tag != nil {
			if len(tagTypes) == 0 || len(valTypes) == 0 {
				return fmt.Errorf("switch case value has unknown type")
			}
			if !tc.IsTypeCompatible(valTypes[0], tagTypes[0]) && !tc.IsTypeCompatible(tagTypes[0], valTypes[0]) {
				return fmt.Errorf("switch case value type %s incompatible with switch tag type %s",
					valTypes[0].Ident, tagTypes[0].Ident)
			}
		} else {
			if len(valTypes) == 0 || !tc.IsTypeCompatible(valTypes[0], ast.TypeNode{Ident: ast.TypeBool}) {
				return fmt.Errorf("switch case condition must be Bool, got %s", valTypes[0].Ident)
			}
		}
		if key, ok := switchCaseValueKey(val); ok {
			if _, dup := seenCaseKeys[key]; dup {
				return fmt.Errorf("duplicate case %s in switch", keyDisplay(key, val))
			}
			seenCaseKeys[key] = struct{}{}
		}
	}
	return tc.inferSwitchClauseBody(clause.Body)
}

func (tc *TypeChecker) inferSwitchClauseBody(body []ast.Node) error {
	for _, node := range body {
		if _, ok := node.(ast.FallthroughNode); ok {
			if tc.switchDepth == 0 {
				return fmt.Errorf("fallthrough statement not inside a switch")
			}
			continue
		}
		if _, err := tc.inferNodeType(node); err != nil {
			return err
		}
	}
	return nil
}

func switchCaseValueKey(expr ast.ExpressionNode) (string, bool) {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		return fmt.Sprintf("int:%d", e.Value), true
	case ast.StringLiteralNode:
		return fmt.Sprintf("string:%q", e.Value), true
	case ast.BoolLiteralNode:
		return fmt.Sprintf("bool:%t", e.Value), true
	case ast.RuneLiteralNode:
		return fmt.Sprintf("rune:%d", e.Value), true
	default:
		return "", false
	}
}

func keyDisplay(key string, expr ast.ExpressionNode) string {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		return fmt.Sprintf("%d", e.Value)
	case ast.StringLiteralNode:
		return fmt.Sprintf("%q", e.Value)
	case ast.BoolLiteralNode:
		if e.Value {
			return "true"
		}
		return "false"
	default:
		return key
	}
}

func (tc *TypeChecker) collectSwitchNode(n *ast.SwitchNode) error {
	if n == nil {
		return nil
	}
	tc.log.WithFields(logrus.Fields{
		"function": "collectSwitchNode",
	}).Debug("Storing scope for switch")
	tc.pushScope(n)
	if n.Init != nil {
		if err := tc.collectExplicitTypes(n.Init); err != nil {
			tc.popScope()
			return err
		}
	}
	for _, clause := range n.Clauses {
		for _, node := range clause.Body {
			if err := tc.collectExplicitTypes(node); err != nil {
				tc.popScope()
				return err
			}
		}
	}
	tc.popScope()
	return nil
}
