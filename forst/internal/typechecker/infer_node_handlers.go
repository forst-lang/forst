package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

func (tc *TypeChecker) inferNodeTypeLeaf(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch node.(type) {
	case ast.PackageNode, ast.TypeNode, ast.CommentNode, ast.ImportNode, ast.ImportGroupNode:
		return nil, true, nil
	case ast.FallthroughNode:
		if tc.switchDepth == 0 {
			return nil, true, fmt.Errorf("fallthrough statement not inside a switch")
		}
		return nil, true, nil
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeParams(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch n := node.(type) {
	case ast.SimpleParamNode:
		if n.Type.Assertion != nil {
			inferredType, err := tc.InferAssertionType(n.Type.Assertion, false, "", nil)
			if err != nil {
				return nil, true, err
			}
			return inferredType, true, nil
		}
		if normalized, ok := tc.normalizeGoImportParamType(n.Type); ok {
			return []ast.TypeNode{normalized}, true, nil
		}
		return []ast.TypeNode{n.Type}, true, nil
	case ast.DestructuredParamNode:
		if n.Type.Assertion != nil {
			inferredType, err := tc.InferAssertionType(n.Type.Assertion, false, "", nil)
			if err != nil {
				return nil, true, err
			}
			return inferredType, true, nil
		}
		if normalized, ok := tc.normalizeGoImportParamType(n.Type); ok {
			return []ast.TypeNode{normalized}, true, nil
		}
		return []ast.TypeNode{n.Type}, true, nil
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeExpression(node ast.Node) ([]ast.TypeNode, bool, error) {
	n, ok := node.(ast.ExpressionNode)
	if !ok {
		return nil, false, nil
	}
	tc.log.WithFields(logrus.Fields{
		"function": "inferNodeType",
		"expr":     n.String(),
	}).Debug("Processing expression node")
	inferredType, err := tc.inferExpressionType(n)
	if err != nil {
		return nil, true, err
	}
	tc.storeInferredType(n, inferredType)
	return inferredType, true, nil
}

func (tc *TypeChecker) inferNodeTypeDecl(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch n := node.(type) {
	case ast.AssignmentNode:
		if n.IsPackageLevel {
			if err := tc.ensurePackageLevelVarRegistered(n); err != nil {
				return nil, true, err
			}
		}
		if err := tc.inferAssignmentTypes(n); err != nil {
			return nil, true, err
		}
		return nil, true, nil
	case ast.ConstGroupNode:
		if err := tc.inferConstGroup(n); err != nil {
			return nil, true, err
		}
		return nil, true, nil
	case ast.TypeDefNode:
		return tc.inferNodeTypeDef(n)
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeControl(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch n := node.(type) {
	case ast.IfNode:
		types, err := tc.inferIfStatement(&n)
		return types, true, err
	case *ast.IfNode:
		types, err := tc.inferIfStatement(n)
		return types, true, err
	case *ast.ForNode:
		types, err := tc.inferForNode(n)
		return types, true, err
	case ast.ForNode:
		types, err := tc.inferForNode(&n)
		return types, true, err
	case *ast.SwitchNode:
		types, err := tc.inferSwitchStatement(n)
		return types, true, err
	case ast.SwitchNode:
		types, err := tc.inferSwitchStatement(&n)
		return types, true, err
	case *ast.BreakNode:
		return tc.inferNodeTypeBreak(n)
	case *ast.ContinueNode:
		return tc.inferNodeTypeContinue(n)
	case *ast.GotoNode:
		return tc.inferNodeTypeGoto(n)
	case *ast.LabeledStmtNode:
		if n.Stmt == nil {
			return nil, true, fmt.Errorf("labeled statement missing body")
		}
		types, err := tc.inferNodeType(n.Stmt)
		return types, true, err
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeBreak(n *ast.BreakNode) ([]ast.TypeNode, bool, error) {
	if n.Label != nil {
		if !tc.hasLoopLabel(n.Label.ID) {
			return nil, true, fmt.Errorf("undefined label %q for break", n.Label.ID)
		}
		return nil, true, nil
	}
	if tc.loopDepth == 0 {
		return nil, true, fmt.Errorf("break is not inside a loop")
	}
	return nil, true, nil
}

func (tc *TypeChecker) inferNodeTypeContinue(n *ast.ContinueNode) ([]ast.TypeNode, bool, error) {
	if n.Label != nil {
		if !tc.hasLoopLabel(n.Label.ID) {
			return nil, true, fmt.Errorf("undefined label %q for continue", n.Label.ID)
		}
		return nil, true, nil
	}
	if tc.loopDepth == 0 {
		return nil, true, fmt.Errorf("continue is not inside a loop")
	}
	return nil, true, nil
}

func (tc *TypeChecker) inferNodeTypeGoto(n *ast.GotoNode) ([]ast.TypeNode, bool, error) {
	if n.Label == nil {
		return nil, true, fmt.Errorf("goto requires a label")
	}
	tc.log.WithFields(logrus.Fields{
		"stmt":  "goto",
		"label": n.Label.ID,
	}).Debug("Typechecking goto statement")
	return nil, true, nil
}

func (tc *TypeChecker) inferNodeTypeDeferGo(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch n := node.(type) {
	case *ast.DeferNode:
		fc, ok := n.Call.(ast.FunctionCallNode)
		if !ok {
			return nil, true, fmt.Errorf("defer requires a function or method call")
		}
		if err := validateDeferGoBuiltinRestriction("defer", fc); err != nil {
			return nil, true, err
		}
		if _, err := tc.inferExpressionType(n.Call); err != nil {
			return nil, true, err
		}
		return nil, true, nil
	case *ast.GoStmtNode:
		fc, ok := n.Call.(ast.FunctionCallNode)
		if !ok {
			return nil, true, fmt.Errorf("go requires a function or method call")
		}
		if err := validateDeferGoBuiltinRestriction("go", fc); err != nil {
			return nil, true, err
		}
		if _, err := tc.inferExpressionType(n.Call); err != nil {
			return nil, true, err
		}
		tc.invalidateAfterSpawn(fc)
		return nil, true, nil
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeElseBlock(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch n := node.(type) {
	case *ast.ElseBlockNode:
		if n == nil {
			return nil, true, nil
		}
		tc.pushScope(n)
		for _, child := range n.Body {
			if _, err := tc.inferNodeType(child); err != nil {
				return nil, true, err
			}
		}
		tc.popScope()
		return nil, true, nil
	case ast.ElseBlockNode:
		eb := n
		types, err := tc.inferNodeType(&eb)
		return types, true, err
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeFunction(node ast.Node) ([]ast.TypeNode, bool, error) {
	if _, ok := node.(ast.FunctionNode); ok {
		types, err := tc.inferFunctionNode(node)
		return types, true, err
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeEnsure(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch n := node.(type) {
	case ast.EnsureNode:
		types, err := tc.inferEnsureNode(node)
		return types, true, err
	case ast.UseNode:
		types, err := tc.inferUseNode(n)
		return types, true, err
	case ast.WithNode:
		types, err := tc.inferWithNode(node)
		return types, true, err
	}
	return nil, false, nil
}

func (tc *TypeChecker) inferNodeTypeReturn(node ast.Node) ([]ast.TypeNode, bool, error) {
	n, ok := node.(ast.ReturnNode)
	if !ok {
		return nil, false, nil
	}
	for _, v := range n.Values {
		if _, err := tc.inferExpressionType(v); err != nil {
			return nil, true, err
		}
	}
	if err := tc.checkReturnDisallowedInResultErrBranch(n); err != nil {
		return nil, true, err
	}
	if err := tc.checkMultiValueReturnLegality(n); err != nil {
		return nil, true, err
	}
	return nil, true, nil
}

func (tc *TypeChecker) inferNodeTypeTypeGuard(node ast.Node) ([]ast.TypeNode, bool, error) {
	switch node.(type) {
	case ast.TypeGuardNode, *ast.TypeGuardNode:
		types, err := tc.inferTypeGuardNode(node)
		return types, true, err
	}
	return nil, false, nil
}
