package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// transformStringEnsure transforms a string ensure
func (at *AssertionTransformer) transformStringEnsure(ensure ast.EnsureNode) (goast.Expr, error) {
	result := []goast.Expr{}

	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr

		switch constraint.Name {
		case MinConstraint:
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
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						at.transformer.transformExpression(ensure.Variable),
					},
				},
				Op: token.LSS,
				Y:  at.transformer.transformExpression(arg),
			}
		case MaxConstraint:
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
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						at.transformer.transformExpression(ensure.Variable),
					},
				},
				Op: token.GTR,
				Y:  at.transformer.transformExpression(arg),
			}
		case HasPrefixConstraint:
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
			expr = negateCondition(&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("strings"),
					Sel: goast.NewIdent("HasPrefix"),
				},
				Args: []goast.Expr{
					at.transformer.transformExpression(ensure.Variable),
					at.transformer.transformExpression(arg),
				},
			})
		default:
			return nil, fmt.Errorf("unknown String constraint: %s", constraint.Name)
		}

		result = append(result, expr)
	}
	return disjoin(result), nil
}

// transformIntEnsure transforms an integer assertion
func (at *AssertionTransformer) transformIntEnsure(ensure ast.EnsureNode) (goast.Expr, error) {
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr

		switch constraint.Name {
		case MinConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.LSS,
				Y:  at.transformer.transformExpression(arg),
			}
		case MaxConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.GTR,
				Y:  at.transformer.transformExpression(arg),
			}
		case LessThanConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.GEQ,
				Y:  at.transformer.transformExpression(arg),
			}
		case GreaterThanConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.LEQ,
				Y:  at.transformer.transformExpression(arg),
			}
		default:
			return nil, fmt.Errorf("unknown Int constraint: %s", constraint.Name)
		}

		result = append(result, expr)
	}
	return disjoin(result), nil
}

// transformFloatEnsure transforms a float assertion
func (at *AssertionTransformer) transformFloatEnsure(ensure ast.EnsureNode) (goast.Expr, error) {
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr

		switch constraint.Name {
		case MinConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.LSS,
				Y:  at.transformer.transformExpression(arg),
			}
		case MaxConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.GTR,
				Y:  at.transformer.transformExpression(arg),
			}
		case GreaterThanConstraint:
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
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.LEQ,
				Y:  at.transformer.transformExpression(arg),
			}
		default:
			return nil, fmt.Errorf("unknown Float constraint: %s", constraint.Name)
		}

		result = append(result, expr)
	}
	return disjoin(result), nil
}

// transformBoolEnsure transforms a boolean assertion
func (at *AssertionTransformer) transformBoolEnsure(ensure ast.EnsureNode) (goast.Expr, error) {
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr

		switch constraint.Name {
		case TrueConstraint:
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			expr = negateCondition(at.transformer.transformExpression(ensure.Variable))
		case FalseConstraint:
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			expr = at.transformer.transformExpression(ensure.Variable)
		default:
			return nil, fmt.Errorf("unknown Bool constraint: %s", constraint.Name)
		}

		result = append(result, expr)
	}
	return disjoin(result), nil
}

// transformErrorEnsure transforms an error assertion
func (at *AssertionTransformer) transformErrorEnsure(ensure ast.EnsureNode) (goast.Expr, error) {
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr

		switch constraint.Name {
		case NilConstraint:
			if err := at.validateConstraintArgs(constraint, 0); err != nil {
				return nil, err
			}
			expr = &goast.BinaryExpr{
				X:  at.transformer.transformExpression(ensure.Variable),
				Op: token.NEQ,
				Y:  goast.NewIdent(NilConstant),
			}
		default:
			return nil, fmt.Errorf("unknown Error constraint: %s", constraint.Name)
		}

		result = append(result, expr)
	}
	return disjoin(result), nil
}
