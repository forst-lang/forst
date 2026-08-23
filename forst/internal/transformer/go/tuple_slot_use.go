package transformergo

import (
	"strconv"

	"forst/internal/ast"
	"forst/internal/astwalk"
	goast "go/ast"
	"go/token"
)

// collectTupleIndexUses returns numeric tuple field indices accessed as var.N in body.
func collectTupleIndexUses(body []ast.Node, varName string) map[int]bool {
	used := make(map[int]bool)
	visit := func(expr ast.ExpressionNode) {
		walkExpressionTree(expr, func(e ast.ExpressionNode) {
			fa, ok := e.(ast.FieldAccessNode)
			if !ok {
				return
			}
			vn, ok := fa.Target.(ast.VariableNode)
			if !ok || string(vn.Ident.ID) != varName {
				return
			}
			idx, err := strconv.Atoi(string(fa.Field.ID))
			if err != nil || idx < 0 {
				return
			}
			used[idx] = true
		})
	}
	astwalk.WalkStmts(body, astwalk.StmtVisitor{
		OnAssign: func(a ast.AssignmentNode) bool {
			for _, lv := range a.LValues {
				if e, ok := lv.(ast.ExpressionNode); ok {
					visit(e)
				}
			}
			for _, rv := range a.RValues {
				visit(rv)
			}
			return true
		},
		OnReturn: func(r ast.ReturnNode) bool {
			for _, v := range r.Values {
				visit(v)
			}
			return true
		},
		OnCall: func(c ast.FunctionCallNode) bool {
			for _, arg := range c.Arguments {
				visit(arg)
			}
			return true
		},
	})
	return used
}

func walkExpressionTree(expr ast.ExpressionNode, visit func(ast.ExpressionNode)) {
	if expr == nil {
		return
	}
	visit(expr)
	switch e := expr.(type) {
	case ast.BinaryExpressionNode:
		walkExpressionTree(e.Left, visit)
		walkExpressionTree(e.Right, visit)
	case ast.UnaryExpressionNode:
		walkExpressionTree(e.Operand, visit)
	case ast.IndexExpressionNode:
		walkExpressionTree(e.Target, visit)
		walkExpressionTree(e.Index, visit)
	case ast.SliceExpressionNode:
		walkExpressionTree(e.Target, visit)
		if e.Low != nil {
			walkExpressionTree(e.Low, visit)
		}
		if e.High != nil {
			walkExpressionTree(e.High, visit)
		}
	case ast.FieldAccessNode:
		walkExpressionTree(e.Target, visit)
	case ast.MethodCallNode:
		walkExpressionTree(e.Receiver, visit)
		for _, arg := range e.Arguments {
			walkExpressionTree(arg, visit)
		}
	case ast.FunctionCallNode:
		if e.Callee != nil {
			walkExpressionTree(e.Callee, visit)
		}
		for _, arg := range e.Arguments {
			walkExpressionTree(arg, visit)
		}
	case ast.ReferenceNode:
		if inner, ok := e.Value.(ast.ExpressionNode); ok {
			walkExpressionTree(inner, visit)
		}
	case ast.DereferenceNode:
		if inner, ok := e.Value.(ast.ExpressionNode); ok {
			walkExpressionTree(inner, visit)
		}
	case ast.ShapeNode:
		for _, field := range e.Fields {
			if fv, ok := field.ValueExpression(); ok {
				walkExpressionTree(fv, visit)
			}
		}
	case ast.ArrayLiteralNode:
		for _, el := range e.Value {
			walkExpressionTree(el, visit)
		}
	case ast.MapLiteralNode:
		for _, entry := range e.Entries {
			if kv, ok := entry.Key.(ast.ExpressionNode); ok {
				walkExpressionTree(kv, visit)
			}
			if vv, ok := entry.Value.(ast.ExpressionNode); ok {
				walkExpressionTree(vv, visit)
			}
		}
	case ast.OkExprNode:
		walkExpressionTree(e.Value, visit)
	case ast.ErrExprNode:
		walkExpressionTree(e.Value, visit)
	}
}

// collectResultErrSlotUsed reports whether lowering must bind the error slot for a Result local.
func collectResultErrSlotUsed(body []ast.Node, varName string) bool {
	if collectResultErrSlotUsedStmts(body, varName) {
		return true
	}
	return bodyUsesVarInResultExpandingPrint(body, varName)
}

func collectResultErrSlotUsedStmts(stmts []ast.Node, varName string) bool {
	for _, n := range stmts {
		switch node := n.(type) {
		case ast.EnsureNode:
			if string(node.Variable.Ident.ID) == varName {
				return true
			}
		case ast.IfNode:
			if resultDiscriminatorUsesVar(node.Condition, varName) {
				return true
			}
			if isOkDiscriminator(node.Condition, varName) && node.Else != nil &&
				bodyReferencesVariable(node.Else.Body, varName) {
				return true
			}
			if collectResultErrSlotUsedStmts(node.Body, varName) {
				return true
			}
			for _, elif := range node.ElseIfs {
				if resultDiscriminatorUsesVar(elif.Condition, varName) {
					return true
				}
				if collectResultErrSlotUsedStmts(elif.Body, varName) {
					return true
				}
			}
			if node.Else != nil && collectResultErrSlotUsedStmts(node.Else.Body, varName) {
				return true
			}
		case *ast.IfNode:
			if resultDiscriminatorUsesVar(node.Condition, varName) {
				return true
			}
			if isOkDiscriminator(node.Condition, varName) && node.Else != nil &&
				bodyReferencesVariable(node.Else.Body, varName) {
				return true
			}
			if collectResultErrSlotUsedStmts(node.Body, varName) {
				return true
			}
			for _, elif := range node.ElseIfs {
				if resultDiscriminatorUsesVar(elif.Condition, varName) {
					return true
				}
				if collectResultErrSlotUsedStmts(elif.Body, varName) {
					return true
				}
			}
			if node.Else != nil && collectResultErrSlotUsedStmts(node.Else.Body, varName) {
				return true
			}
		case ast.ForNode, *ast.ForNode:
			var forBody []ast.Node
			switch fn := node.(type) {
			case ast.ForNode:
				forBody = fn.Body
			case *ast.ForNode:
				forBody = fn.Body
			}
			if collectResultErrSlotUsedStmts(forBody, varName) {
				return true
			}
		}
	}
	return false
}

func resultDiscriminatorUsesVar(cond ast.Node, varName string) bool {
	expr, ok := cond.(ast.ExpressionNode)
	if !ok {
		return false
	}
	bin, ok := expr.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		return false
	}
	vn, ok := bin.Left.(ast.VariableNode)
	if !ok || string(vn.Ident.ID) != varName {
		return false
	}
	asn, ok := bin.Right.(ast.AssertionNode)
	if !ok {
		return false
	}
	for _, c := range asn.Constraints {
		if c.Name == "Ok" || c.Name == "Err" {
			return true
		}
	}
	return false
}

func isOkDiscriminator(cond ast.Node, varName string) bool {
	expr, ok := cond.(ast.ExpressionNode)
	if !ok {
		return false
	}
	bin, ok := expr.(ast.BinaryExpressionNode)
	if !ok || bin.Operator != ast.TokenIs {
		return false
	}
	vn, ok := bin.Left.(ast.VariableNode)
	if !ok || string(vn.Ident.ID) != varName {
		return false
	}
	asn, ok := bin.Right.(ast.AssertionNode)
	if !ok {
		return false
	}
	for _, c := range asn.Constraints {
		if c.Name == "Ok" {
			return true
		}
	}
	return false
}

func bodyUsesVarInResultExpandingPrint(body []ast.Node, varName string) bool {
	found := false
	astwalk.WalkStmts(body, astwalk.StmtVisitor{
		OnCall: func(c ast.FunctionCallNode) bool {
			if !isPrintLikeBuiltinCall(c.Function) {
				return true
			}
			for _, arg := range c.Arguments {
				if vn, ok := arg.(ast.VariableNode); ok && string(vn.Ident.ID) == varName {
					found = true
					return false
				}
			}
			return true
		},
	})
	return found
}

func bodyReferencesVariable(body []ast.Node, varName string) bool {
	found := false
	visit := func(expr ast.ExpressionNode) {
		walkExpressionTree(expr, func(e ast.ExpressionNode) {
			if vn, ok := e.(ast.VariableNode); ok && string(vn.Ident.ID) == varName {
				found = true
			}
		})
	}
	astwalk.WalkStmts(body, astwalk.StmtVisitor{
		OnAssign: func(a ast.AssignmentNode) bool {
			for _, rv := range a.RValues {
				visit(rv)
			}
			return true
		},
		OnReturn: func(r ast.ReturnNode) bool {
			for _, v := range r.Values {
				visit(v)
			}
			return true
		},
		OnCall: func(c ast.FunctionCallNode) bool {
			for _, arg := range c.Arguments {
				visit(arg)
			}
			return true
		},
	})
	return found
}

// collectVariableAnyUse reports whether varName appears in any expression in body.
func collectVariableAnyUse(body []ast.Node, varName string) bool {
	return bodyReferencesVariable(body, varName)
}

// assignOpForMultiValueLHS picks := vs = for multi-value assignment; all-blank LHS must use =.
func assignOpForMultiValueLHS(isShort bool, lhs []goast.Expr) token.Token {
	if !isShort {
		return token.ASSIGN
	}
	for _, l := range lhs {
		if ident, ok := l.(*goast.Ident); !ok || ident.Name == "_" {
			continue
		}
		return token.DEFINE
	}
	return token.ASSIGN
}
