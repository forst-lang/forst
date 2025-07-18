package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// Constraint types
type ConstraintHandler func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error)

// Map of builtin types to their constraints and handlers
var builtinConstraints = map[ast.TypeIdent]map[BuiltinConstraint]ConstraintHandler{
	ast.TypePointer: {
		NilConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.NEQ,
				Y:  goast.NewIdent(NilConstant),
			}, nil
		},
		PresentConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.EQL,
				Y:  goast.NewIdent(NilConstant),
			}, nil
		},
	},
	ast.TypeString: {
		MinConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						variableExpr,
					},
				},
				Op: token.LSS,
				Y:  argExpr,
			}, nil
		},
		MaxConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						variableExpr,
					},
				},
				Op: token.GTR,
				Y:  argExpr,
			}, nil
		},
		HasPrefixConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			// Ensure the strings import is present
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return negateCondition(&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("strings"),
					Sel: goast.NewIdent("HasPrefix"),
				},
				Args: []goast.Expr{
					variableExpr,
					argExpr,
				},
			}), nil
		},
		ContainsConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			// Ensure the strings import is present
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return negateCondition(&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("strings"),
					Sel: goast.NewIdent("Contains"),
				},
				Args: []goast.Expr{
					variableExpr,
					argExpr,
				},
			}), nil
		},
		NotEmptyConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						variableExpr,
					},
				},
				Op: token.EQL,
				Y:  &goast.BasicLit{Kind: token.INT, Value: "0"},
			}, nil
		},
		ValidConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			// For now, just return a simple condition that will always be false
			// This simulates a validation that always fails
			return goast.NewIdent("false"), nil
		},
	},
	ast.TypeInt: {
		MinConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.LSS,
				Y:  argExpr,
			}, nil
		},
		MaxConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.GTR,
				Y:  argExpr,
			}, nil
		},
		LessThanConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.GEQ,
				Y:  argExpr,
			}, nil
		},
		GreaterThanConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.LEQ,
				Y:  argExpr,
			}, nil
		},
	},
	ast.TypeFloat: {
		MinConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.LSS,
				Y:  argExpr,
			}, nil
		},
		MaxConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.GTR,
				Y:  argExpr,
			}, nil
		},
		GreaterThanConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
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
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			argExpr, err := at.transformer.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.LEQ,
				Y:  argExpr,
			}, nil
		},
	},
	ast.TypeBool: {
		TrueConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return negateCondition(variableExpr), nil
		},
		FalseConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return variableExpr, nil
		},
	},
	ast.TypeError: {
		NilConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.NEQ,
				Y:  goast.NewIdent(NilConstant),
			}, nil
		},
		PresentConstraint: func(at *AssertionTransformer, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			variableExpr, err := at.transformer.transformExpression(variable)
			if err != nil {
				return nil, err
			}
			return &goast.BinaryExpr{
				X:  variableExpr,
				Op: token.EQL,
				Y:  goast.NewIdent(NilConstant),
			}, nil
		},
	},
}

// TransformBuiltinConstraint transforms a builtin constraint
func (at *AssertionTransformer) TransformBuiltinConstraint(typeIdent ast.TypeIdent, variable ast.VariableNode, constraint ast.ConstraintNode) (goast.Expr, error) {
	handlerMap, ok := builtinConstraints[typeIdent]
	if !ok {
		return nil, fmt.Errorf("unknown typeIdent %s for built-in constraints: %s", typeIdent, constraint.Name)
	}
	handler, ok := handlerMap[BuiltinConstraint(constraint.Name)]
	if !ok {
		return nil, fmt.Errorf("unknown constraint: %s", constraint.Name)
	}
	return handler(at, variable, constraint)
}
