package transformergo

import (
	"errors"
	"fmt"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

// errMissingConstraintArgValue is returned when an Ok/Err constraint argument omits a value.
var errMissingConstraintArgValue = errors.New("missing value argument for constraint argument")

// transformIfIsCondition builds a Go bool for `left is <assertion>` when the RHS is an AssertionNode.
// The typechecker already validated the branch; we emit an expression that evaluates the subject
// (for side effects) and returns true. Constrained assertions need dedicated runtime checks (TODO).
func (t *Transformer) transformIfIsCondition(left ast.ExpressionNode, assertion *ast.AssertionNode) (goast.Expr, error) {
	if assertion == nil {
		return nil, fmt.Errorf("if-is: nil assertion")
	}
	if len(assertion.Constraints) == 1 && assertion.BaseType == nil {
		c := assertion.Constraints[0]
		if c.Name == "Ok" || c.Name == "Err" {
			return t.transformResultIsDiscriminator(left, c)
		}
	}
	leftExpr, err := t.transformExpression(left)
	if err != nil {
		return nil, err
	}
	if len(assertion.Constraints) > 0 {
		return nil, fmt.Errorf("if-is with assertion constraints is not yet supported in Go codegen")
	}
	fn := &goast.FuncLit{
		Type: &goast.FuncType{
			Results: &goast.FieldList{
				List: []*goast.Field{{Type: goast.NewIdent("bool")}},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.AssignStmt{
					Lhs: []goast.Expr{goast.NewIdent("_")},
					Tok: token.ASSIGN,
					Rhs: []goast.Expr{leftExpr},
				},
				&goast.ReturnStmt{
					Results: []goast.Expr{goast.NewIdent("true")},
				},
			},
		},
	}
	return &goast.CallExpr{Fun: fn}, nil
}

func expressionFromConstraintArg(arg ast.ConstraintArgumentNode) (ast.ExpressionNode, error) {
	if arg.Value == nil {
		return nil, errMissingConstraintArgValue
	}
	return *arg.Value, nil
}

func (t *Transformer) goResultErrIdentForExpr(left ast.ExpressionNode) (goast.Expr, error) {
	vn, ok := left.(ast.VariableNode)
	if !ok {
		return nil, fmt.Errorf("if-is: Result Ok/Err requires a simple variable subject")
	}
	name := string(vn.Ident.ID)
	if isDotQualifiedVariable(vn) {
		if t.compoundVarDeclaresResultField(vn) {
			return t.goResultErrIdentForCompoundField(vn)
		}
		return nil, fmt.Errorf("if-is: Result Ok/Err on %q is not a lowered Result struct field", name)
	}
	if t.resultLocalSplit != nil {
		if split, ok := t.resultLocalSplit[name]; ok && split.errGoName != "" {
			return goast.NewIdent(split.errGoName), nil
		}
	}
	return nil, fmt.Errorf("if-is: Result Ok/Err requires a Result-bound variable (missing local split for %q)", name)
}

// goExprErrFailureTypeAssertion builds a bool expression for `Err(T)` when T is a type argument:
// failure value is non-nil and type-asserts to T (comma-ok), for codegen when no runtime value is given.
func (t *Transformer) goExprErrFailureTypeAssertion(errExpr goast.Expr, discriminant *ast.TypeNode) (goast.Expr, error) {
	if discriminant == nil {
		return nil, fmt.Errorf("if-is: Err type discriminator needs a type")
	}
	goTy, err := t.transformType(*discriminant)
	if err != nil {
		return nil, err
	}
	// func() bool {
	//   if errExpr == nil { return false }
	//   _, ok := errExpr.(T)
	//   return ok
	// }()
	ifNil := &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: errExpr, Op: token.EQL, Y: goast.NewIdent("nil")},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("false")}},
		}},
	}
	assign := &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent("_"), goast.NewIdent("ok")},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{&goast.TypeAssertExpr{X: errExpr, Type: goTy}},
	}
	retOk := &goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("ok")}}
	fn := &goast.FuncLit{
		Type: &goast.FuncType{
			Results: &goast.FieldList{
				List: []*goast.Field{{Type: goast.NewIdent("bool")}},
			},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{ifNil, assign, retOk}},
	}
	return &goast.CallExpr{Fun: fn}, nil
}

// transformResultIsDiscriminator lowers `x is Ok(...)` / `Err(...)` for Result lowered to (succ, err).
func (t *Transformer) transformResultIsDiscriminator(left ast.ExpressionNode, c ast.ConstraintNode) (goast.Expr, error) {
	errExpr, err := t.goResultErrIdentForExpr(left)
	if err != nil {
		return nil, err
	}
	switch c.Name {
	case "Ok":
		succExpr, err := t.goResultSuccessValueExprForOkDiscriminator(left)
		if err != nil {
			return nil, err
		}
		check := goast.Expr(&goast.BinaryExpr{
			X: errExpr, Op: token.EQL, Y: goast.NewIdent("nil"),
		})
		if len(c.Args) == 1 {
			argExpr, err := expressionFromConstraintArg(c.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err := t.transformExpression(argExpr)
			if err != nil {
				return nil, err
			}
			eq := &goast.BinaryExpr{X: succExpr, Op: token.EQL, Y: arg}
			check = &goast.BinaryExpr{X: check, Op: token.LAND, Y: eq}
		}
		return check, nil
	case "Err":
		if len(c.Args) == 0 {
			return &goast.BinaryExpr{X: errExpr, Op: token.NEQ, Y: goast.NewIdent("nil")}, nil
		}
		if len(c.Args) == 1 {
			if c.Args[0].Type != nil {
				return t.goExprErrFailureTypeAssertion(errExpr, c.Args[0].Type)
			}
			argExpr, err := expressionFromConstraintArg(c.Args[0])
			if err != nil {
				return nil, err
			}
			arg, err := t.transformExpression(argExpr)
			if err != nil {
				return nil, err
			}
			ne := &goast.BinaryExpr{X: errExpr, Op: token.NEQ, Y: goast.NewIdent("nil")}
			eq := &goast.BinaryExpr{X: errExpr, Op: token.EQL, Y: arg}
			return &goast.BinaryExpr{X: ne, Op: token.LAND, Y: eq}, nil
		}
		return nil, errors.New("if-is: Err(...) expects at most one argument")
	default:
		return nil, errors.New("internal: not Ok/Err discriminator")
	}
}
