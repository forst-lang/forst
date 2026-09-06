package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"sync"
)

// ConstraintHandler returns a Go bool expression that is true when the constraint holds
// (success polarity). Ensure and type-guard emitters negate where they need a failure branch.
type ConstraintHandler func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error)

var (
	builtinConstraintsOnce sync.Once
	builtinConstraints     map[ast.TypeIdent]map[BuiltinConstraint]ConstraintHandler
)

func lenCall(x goast.Expr) *goast.CallExpr {
	return &goast.CallExpr{Fun: goast.NewIdent("len"), Args: []goast.Expr{x}}
}

func runeCountCall(at *AssertionTransformer, x goast.Expr) *goast.CallExpr {
	at.transformer.Output.EnsureImport("unicode/utf8")
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("utf8"),
			Sel: goast.NewIdent("RuneCountInString"),
		},
		Args: []goast.Expr{x},
	}
}

func binCmp(x goast.Expr, op token.Token, y goast.Expr) *goast.BinaryExpr {
	return &goast.BinaryExpr{X: x, Op: op, Y: y}
}

func nilCmp(x goast.Expr, op token.Token) *goast.BinaryExpr {
	return binCmp(x, op, goast.NewIdent(NilConstant))
}

func zeroLit() *goast.BasicLit {
	return &goast.BasicLit{Kind: token.INT, Value: "0"}
}

// lengthBoundHandlers emit Min/Max/NotEmpty against len(subject) (arrays, maps, bytes).
func lengthBoundHandlers() map[BuiltinConstraint]ConstraintHandler {
	return map[BuiltinConstraint]ConstraintHandler{
		MinConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 1); err != nil {
				return nil, err
			}
			arg, err := at.expectValue(&constraint.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err = expectIntLiteral(arg)
			if err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return binCmp(lenCall(variableExpr), token.GEQ, argExpr), nil
		},
		MaxConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 1); err != nil {
				return nil, err
			}
			arg, err := at.expectValue(&constraint.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err = expectIntLiteral(arg)
			if err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return binCmp(lenCall(variableExpr), token.LEQ, argExpr), nil
		},
		NotEmptyConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			return binCmp(lenCall(variableExpr), token.NEQ, zeroLit()), nil
		},
	}
}

func nilPresentHandlers() map[BuiltinConstraint]ConstraintHandler {
	return map[BuiltinConstraint]ConstraintHandler{
		NilConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			return nilCmp(variableExpr, token.EQL), nil
		},
		PresentConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			return nilCmp(variableExpr, token.NEQ), nil
		},
	}
}

func numericBoundHandlers() map[BuiltinConstraint]ConstraintHandler {
	return map[BuiltinConstraint]ConstraintHandler{
		MinConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 1); err != nil {
				return nil, err
			}
			arg, err := at.expectValue(&constraint.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err = expectNumberLiteral(arg)
			if err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return binCmp(variableExpr, token.GEQ, argExpr), nil
		},
		MaxConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 1); err != nil {
				return nil, err
			}
			arg, err := at.expectValue(&constraint.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err = expectNumberLiteral(arg)
			if err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return binCmp(variableExpr, token.LEQ, argExpr), nil
		},
		LessThanConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 1); err != nil {
				return nil, err
			}
			arg, err := at.expectValue(&constraint.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err = expectNumberLiteral(arg)
			if err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return binCmp(variableExpr, token.LSS, argExpr), nil
		},
		GreaterThanConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 1); err != nil {
				return nil, err
			}
			arg, err := at.expectValue(&constraint.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err = expectNumberLiteral(arg)
			if err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(subject)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return binCmp(variableExpr, token.GTR, argExpr), nil
		},
	}
}

func initBuiltinConstraints() {
	length := lengthBoundHandlers()
	nilPresent := nilPresentHandlers()
	numeric := numericBoundHandlers()

	floatHandlers := make(map[BuiltinConstraint]ConstraintHandler, len(numeric)+1)
	for k, v := range numeric {
		floatHandlers[k] = v
	}
	floatHandlers[FiniteConstraint] = func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
		if err := at.validateConstraintArgs(constraint, 0); err != nil {
			return nil, err
		}
		at.transformer.Output.EnsureImport("math")
		variableExpr, err := at.transformer.transformExpression(subject)
		if err != nil {
			return nil, err
		}
		isInf := &goast.CallExpr{
			Fun: &goast.SelectorExpr{X: goast.NewIdent("math"), Sel: goast.NewIdent("IsInf")},
			Args: []goast.Expr{
				variableExpr,
				zeroLit(),
			},
		}
		isNaN := &goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: goast.NewIdent("math"), Sel: goast.NewIdent("IsNaN")},
			Args: []goast.Expr{variableExpr},
		}
		return &goast.BinaryExpr{
			X:  negateCondition(isInf),
			Op: token.LAND,
			Y:  negateCondition(isNaN),
		}, nil
	}

	builtinConstraints = map[ast.TypeIdent]map[BuiltinConstraint]ConstraintHandler{
		ast.TypePointer: nilPresent,
		ast.TypeError:   nilPresent,
		ast.TypeArray:   length,
		ast.TypeMap:     length,
		ast.TypeBytes:   length,
		ast.TypeInt:     numeric,
		ast.TypeFloat:   floatHandlers,
		ast.TypeString: {
			MinConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 1); err != nil {
					return nil, err
				}
				variableExpr, err := at.transformStringBuiltinVariable(subject)
				if err != nil {
					return nil, err
				}
				argExpr, err := at.constraintArgAsExpr(constraint.Args[0])
				if err != nil {
					return nil, err
				}
				return binCmp(runeCountCall(at, variableExpr), token.GEQ, argExpr), nil
			},
			MaxConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 1); err != nil {
					return nil, err
				}
				variableExpr, err := at.transformStringBuiltinVariable(subject)
				if err != nil {
					return nil, err
				}
				argExpr, err := at.constraintArgAsExpr(constraint.Args[0])
				if err != nil {
					return nil, err
				}
				return binCmp(runeCountCall(at, variableExpr), token.LEQ, argExpr), nil
			},
			MinBytesConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 1); err != nil {
					return nil, err
				}
				variableExpr, err := at.transformStringBuiltinVariable(subject)
				if err != nil {
					return nil, err
				}
				argExpr, err := at.constraintArgAsExpr(constraint.Args[0])
				if err != nil {
					return nil, err
				}
				return binCmp(lenCall(variableExpr), token.GEQ, argExpr), nil
			},
			MaxBytesConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 1); err != nil {
					return nil, err
				}
				variableExpr, err := at.transformStringBuiltinVariable(subject)
				if err != nil {
					return nil, err
				}
				argExpr, err := at.constraintArgAsExpr(constraint.Args[0])
				if err != nil {
					return nil, err
				}
				return binCmp(lenCall(variableExpr), token.LEQ, argExpr), nil
			},
			HasPrefixConstraint: stringHasAffixHandler("HasPrefix"),
			HasSuffixConstraint: stringHasAffixHandler("HasSuffix"),
			ContainsConstraint:  stringHasAffixHandler("Contains"),
			NotEmptyConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 0); err != nil {
					return nil, err
				}
				variableExpr, err := at.transformStringBuiltinVariable(subject)
				if err != nil {
					return nil, err
				}
				return binCmp(lenCall(variableExpr), token.NEQ, zeroLit()), nil
			},
		},
		ast.TypeBool: {
			TrueConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 0); err != nil {
					return nil, err
				}
				return at.transformer.transformExpression(subject)
			},
			FalseConstraint: func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
				if err := at.validateConstraintArgs(constraint, 0); err != nil {
					return nil, err
				}
				variableExpr, err := at.transformer.transformExpression(subject)
				if err != nil {
					return nil, err
				}
				return negateCondition(variableExpr), nil
			},
		},
	}
}

func stringHasAffixHandler(method string) ConstraintHandler {
	return func(at *AssertionTransformer, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
		at.transformer.Output.EnsureImport("strings")
		if err := at.validateConstraintArgs(constraint, 1); err != nil {
			return nil, err
		}
		arg, err := at.expectValue(&constraint.Args[0])
		if err != nil {
			return nil, err
		}
		arg, err = expectStringLiteral(arg)
		if err != nil {
			return nil, err
		}
		variableExpr, err := at.transformStringBuiltinVariable(subject)
		if err != nil {
			return nil, err
		}
		argExpr, err := at.transformer.transformExpression(arg)
		if err != nil {
			return nil, err
		}
		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("strings"),
				Sel: goast.NewIdent(method),
			},
			Args: []goast.Expr{variableExpr, argExpr},
		}, nil
	}
}

// TransformBuiltinConstraint transforms a builtin constraint into a success-polarity Go bool expr.
func (at *AssertionTransformer) TransformBuiltinConstraint(typeIdent ast.TypeIdent, subject ast.ExpressionNode, constraint ast.ConstraintNode) (goast.Expr, error) {
	builtinConstraintsOnce.Do(initBuiltinConstraints)
	handlerMap, ok := builtinConstraints[typeIdent]
	if !ok {
		return nil, fmt.Errorf("unknown typeIdent %s for built-in constraints: %s", typeIdent, constraint.Name)
	}
	handler, ok := handlerMap[BuiltinConstraint(constraint.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown constraint: %s", constraint.Name)
	}
	return handler(at, subject, constraint)
}
