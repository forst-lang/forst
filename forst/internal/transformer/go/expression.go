package transformergo

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func (t *Transformer) transformExpression(expr ast.ExpressionNode) (goast.Expr, error) {
	switch e := expr.(type) {
	case ast.IntLiteralNode:
		return &goast.BasicLit{
			Kind:  token.INT,
			Value: strconv.FormatInt(e.Value, 10),
		}, nil
	case ast.FloatLiteralNode:
		return &goast.BasicLit{
			Kind:  token.FLOAT,
			Value: strconv.FormatFloat(e.Value, 'f', -1, 64),
		}, nil
	case ast.StringLiteralNode:
		return &goast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(e.Value),
		}, nil
	case ast.BoolLiteralNode:
		if e.Value {
			return goast.NewIdent("true"), nil
		}
		return goast.NewIdent("false"), nil
	case ast.NilLiteralNode:
		return goast.NewIdent("nil"), nil
	case ast.ArrayLiteralNode:
		elts := make([]goast.Expr, 0, len(e.Value))
		for _, item := range e.Value {
			ex, err := t.transformExpression(item.(ast.ExpressionNode))
			if err != nil {
				return nil, err
			}
			elts = append(elts, ex)
		}
		elemType := ast.TypeNode{Ident: ast.TypeInt}
		if hash, err := t.TypeChecker.Hasher.HashNode(e); err == nil {
			if types, ok := t.TypeChecker.Types[hash]; ok && len(types) == 1 &&
				types[0].Ident == ast.TypeArray && len(types[0].TypeParams) > 0 {
				elemType = types[0].TypeParams[0]
			}
		}
		eltGo, err := t.transformType(elemType)
		if err != nil {
			return nil, err
		}
		return &goast.CompositeLit{
			Type: &goast.ArrayType{Elt: eltGo},
			Elts: elts,
		}, nil
	case ast.MapLiteralNode:
		if e.Type.Ident != ast.TypeMap || len(e.Type.TypeParams) != 2 {
			return nil, fmt.Errorf("map literal: invalid type %v", e.Type)
		}
		keyGo, err := t.transformType(e.Type.TypeParams[0])
		if err != nil {
			return nil, err
		}
		valGo, err := t.transformType(e.Type.TypeParams[1])
		if err != nil {
			return nil, err
		}
		elts := make([]goast.Expr, 0, len(e.Entries))
		for _, ent := range e.Entries {
			kx, err := t.transformExpression(ent.Key)
			if err != nil {
				return nil, err
			}
			vx, err := t.transformExpression(ent.Value)
			if err != nil {
				return nil, err
			}
			elts = append(elts, &goast.KeyValueExpr{Key: kx, Value: vx})
		}
		return &goast.CompositeLit{
			Type: &goast.MapType{Key: keyGo, Value: valGo},
			Elts: elts,
		}, nil
	case ast.UnaryExpressionNode:
		op, err := t.transformOperator(e.Operator)
		if err != nil {
			return nil, err
		}
		expr, err := t.transformExpression(e.Operand)
		if err != nil {
			return nil, err
		}
		return &goast.UnaryExpr{
			Op: op,
			X:  expr,
		}, nil
	case ast.BinaryExpressionNode:
		if e.Operator == ast.TokenIs {
			var asn *ast.AssertionNode
			switch r := e.Right.(type) {
			case ast.AssertionNode:
				cp := r
				asn = &cp
			case *ast.AssertionNode:
				asn = r
			}
			if asn != nil {
				return t.transformIfIsCondition(e.Left, asn)
			}
		}
		left, err := t.transformExpression(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := t.transformExpression(e.Right)
		if err != nil {
			return nil, err
		}
		op, err := t.transformOperator(e.Operator)
		if err != nil {
			return nil, err
		}
		return &goast.BinaryExpr{
			X:  left,
			Op: op,
			Y:  right,
		}, nil
	case ast.VariableNode:
		// Support field access: split by '.' and capitalize each field after the first
		parts := strings.Split(e.GetIdent(), ".")
		if len(parts) == 1 {
			if t != nil && t.resultLocalSplit != nil {
				if split, ok := t.resultLocalSplit[parts[0]]; ok && len(split.successGoNames) > 1 {
					return nil, fmt.Errorf(
						"codegen: %q is a Result whose success lowers to multiple Go values %v; use multiple left-hand names (e.g. a, b, err := pkg.F()) instead of a single binding",
						parts[0], split.successGoNames)
				}
				if split, ok := t.resultLocalSplit[parts[0]]; ok {
					if expr, ok2 := t.goExprForNarrowedResultSplitLocal(e, split); ok2 {
						return expr, nil
					}
				}
			}
			return &goast.Ident{Name: parts[0]}, nil
		}
		var sel goast.Expr = goast.NewIdent(parts[0])
		for _, field := range parts[1:] {
			fieldName := field
			if t.ExportReturnStructFields {
				fieldName = capitalizeFirst(field)
			}
			sel = &goast.SelectorExpr{
				X:   sel,
				Sel: goast.NewIdent(fieldName),
			}
		}
		sel = t.appendResultStoragePayloadSelector(e, sel)
		return sel, nil
	case ast.IndexExpressionNode:
		tgt, err := t.transformExpression(e.Target)
		if err != nil {
			return nil, err
		}
		idx, err := t.transformExpression(e.Index)
		if err != nil {
			return nil, err
		}
		return &goast.IndexExpr{
			X:     tgt,
			Index: idx,
		}, nil

	case ast.FunctionCallNode:
		if isPrintLikeBuiltinCall(e.Function) {
			args, err := t.transformPrintBuiltinCallArgs(e.Arguments)
			if err != nil {
				return nil, err
			}
			return &goast.CallExpr{
				Fun:  goFunExprFromForstCallIdent(e.Function),
				Args: args,
			}, nil
		}
		// Look up parameter types for the function
		paramTypes := make([]ast.TypeNode, len(e.Arguments))
		if sig, ok := t.TypeChecker.Functions[e.Function.ID]; ok && len(sig.Parameters) == len(e.Arguments) {
			for i, param := range sig.Parameters {
				if param.Type.Ident == ast.TypeAssertion && param.Type.Assertion != nil {
					inferredTypes, err := t.TypeChecker.InferAssertionType(param.Type.Assertion, false, "", nil)
					if err == nil && len(inferredTypes) > 0 {
						paramTypes[i] = inferredTypes[0]
					} else {
						paramTypes[i] = param.Type
					}
				} else {
					paramTypes[i] = param.Type
				}
			}
		}
		args := make([]goast.Expr, len(e.Arguments))
		for i, arg := range e.Arguments {
			if shapeArg, ok := arg.(ast.ShapeNode); ok && paramTypes[i].Ident != ast.TypeImplicit {
				// Use the unified helper to determine the expected type
				context := &ShapeContext{
					ExpectedType:   &paramTypes[i],
					FunctionName:   string(e.Function.ID),
					ParameterIndex: i,
				}
				expectedTypeForShape := t.getExpectedTypeForShape(&shapeArg, context)
				argExpr, err := t.transformShapeNodeWithExpectedType(&shapeArg, expectedTypeForShape)
				if err != nil {
					return nil, err
				}
				args[i] = argExpr
			} else {
				argExpr, err := t.transformExpression(arg)
				if err != nil {
					return nil, err
				}
				args[i] = argExpr
			}
		}
		return &goast.CallExpr{
			Fun:  t.goFunExprFromForstCallIdentWithNarrowing(e.Function),
			Args: args,
		}, nil
	case ast.ReferenceNode:
		expr, err := t.transformExpression(e.Value)
		if err != nil {
			return nil, err
		}
		// Check if the inner expression is a struct literal (CompositeLit)
		if _, isStructLiteral := expr.(*goast.CompositeLit); isStructLiteral {
			// For struct literals, just return the struct literal
			// The outer context will handle adding the & if needed
			return expr, nil
		}
		// For other expressions, add the & operator
		return &goast.UnaryExpr{
			Op: token.AND,
			X:  expr,
		}, nil
	case ast.DereferenceNode:
		expr, err := t.transformExpression(e.Value)
		if err != nil {
			return nil, err
		}
		return &goast.UnaryExpr{
			Op: token.MUL,
			X:  expr,
		}, nil
	case ast.ShapeNode:
		// Use the unified helper to determine the expected type
		context := &ShapeContext{}
		expectedType := t.getExpectedTypeForShape(&e, context)
		// Always use the unified aliasing logic for shape literals
		return t.transformShapeNodeWithExpectedType(&e, expectedType)

	case ast.OkExprNode, ast.ErrExprNode:
		return nil, fmt.Errorf("Ok(...) and Err(...) are lowered in return statements only (not as a value expression)")
	}

	return nil, fmt.Errorf("unsupported expression type: %s", reflect.TypeOf(expr).String())
}
