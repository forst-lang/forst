package transformer_go

import (
	"forst/pkg/ast"
	goast "go/ast"
	"go/token"
)

const (
	MIN_CONSTRAINT        = "Min"
	MAX_CONSTRAINT        = "Max"
	HAS_PREFIX_CONSTRAINT = "HasPrefix"
	TRUE_CONSTRAINT       = "True"
	FALSE_CONSTRAINT      = "False"
	NIL_CONSTRAINT        = "Nil"
)

const (
	BOOL_CONSTANT_TRUE  = "true"
	BOOL_CONSTANT_FALSE = "false"
	NIL_CONSTANT        = "nil"
)

func negateCondition(condition goast.Expr) goast.Expr {
	return &goast.UnaryExpr{
		Op: token.NOT,
		X:  condition,
	}
}

func any(conditions []goast.Expr) goast.Expr {
	if len(conditions) == 0 {
		return &goast.Ident{Name: BOOL_CONSTANT_FALSE}
	}
	combined := conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined = &goast.BinaryExpr{
			X:  combined,
			Op: token.LOR,
			Y:  conditions[i],
		}
	}
	return combined
}

func (t *Transformer) transformStringAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}

	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						goast.NewIdent(ensure.Variable.Id()),
					},
				},
				Op: token.LSS,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						goast.NewIdent(ensure.Variable.Id()),
					},
				},
				Op: token.GTR,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "HasPrefix":
			if len(constraint.Args) != 1 {
				panic("HasPrefix constraint requires 1 argument")
			}
			expr = negateCondition(&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("strings"),
					Sel: goast.NewIdent("HasPrefix"),
				},
				Args: []goast.Expr{
					goast.NewIdent(ensure.Variable.Id()),
					transformExpression(constraint.Args[0]),
				},
			})
		default:
			panic("Unknown String constraint: " + constraint.Name)
		}

		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) transformIntAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.LSS,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.GTR,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "LessThan":
			if len(constraint.Args) != 1 {
				panic("LessThan constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.GEQ,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "GreaterThan":
			if len(constraint.Args) != 1 {
				panic("GreaterThan constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.LEQ,
				Y:  transformExpression(constraint.Args[0]),
			}
		default:
			panic("Unknown Int constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) transformFloatAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.LSS,
				Y:  transformExpression(constraint.Args[0]),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.GTR,
				Y:  transformExpression(constraint.Args[0]),
			}
		default:
			panic("Unknown Float constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) transformBoolAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "True":
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.EQL,
				Y:  goast.NewIdent(BOOL_CONSTANT_TRUE),
			}
		case "False":
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.EQL,
				Y:  goast.NewIdent(BOOL_CONSTANT_FALSE),
			}
		default:
			panic("Unknown Bool constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) transformErrorAssertion(ensure ast.EnsureNode) goast.Expr {
	var result []goast.Expr = []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case NIL_CONSTRAINT:
			expr = &goast.BinaryExpr{
				X:  goast.NewIdent(ensure.Variable.Id()),
				Op: token.NEQ,
				Y:  goast.NewIdent(NIL_CONSTANT),
			}
		default:
			panic("Unknown Error constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) getAssertionBaseType(ensure ast.EnsureNode) ast.TypeNode {
	if ensure.Assertion.BaseType != nil {
		return ast.TypeNode{Name: *ensure.Assertion.BaseType}
	}

	assertionType, err := t.TypeChecker.LookupAssertionType(&ensure, t.currentScope)
	if err != nil {
		panic(err)
	}
	return *assertionType
}

// transformEnsure converts a Forst ensure to a Go expression
func (t *Transformer) transformEnsureCondition(ensure ast.EnsureNode) goast.Expr {
	baseType := t.getAssertionBaseType(ensure)

	switch baseType.Name {
	case ast.TypeString:
		return t.transformStringAssertion(ensure)
	case ast.TypeInt:
		return t.transformIntAssertion(ensure)
	case ast.TypeFloat:
		return t.transformFloatAssertion(ensure)
	case ast.TypeBool:
		return t.transformBoolAssertion(ensure)
	case ast.TypeError:
		return t.transformErrorAssertion(ensure)
	default:
		panic("Unknown base type: " + baseType.Name)
	}
}
