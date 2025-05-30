package transformer_go

import (
	"forst/internal/ast"
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

func expectNumberLiteral(arg *ast.ConstraintArgumentNode) *ast.ValueNode {
	if arg == nil {
		panic("Expected an argument")
	}
	if arg.Value == nil {
		panic("Expected argument to be a value")
	}
	if (*arg.Value).Kind() != ast.NodeKindIntLiteral && (*arg.Value).Kind() != ast.NodeKindFloatLiteral {
		panic("Expected value to be a number literal")
	}
	return arg.Value
}

func expectIntLiteral(arg *ast.ConstraintArgumentNode) *ast.ValueNode {
	if arg == nil {
		panic("Expected an argument")
	}
	if arg.Value == nil {
		panic("Expected argument to be a value")
	}
	if (*arg.Value).Kind() != ast.NodeKindIntLiteral {
		panic("Expected value to be an int literal")
	}
	return arg.Value
}

func expectStringLiteral(arg *ast.ConstraintArgumentNode) *ast.ValueNode {
	if arg == nil {
		panic("Expected an argument")
	}
	if arg.Value == nil {
		panic("Expected argument to be a value")
	}
	if (*arg.Value).Kind() != ast.NodeKindStringLiteral {
		panic("Expected value to be a string literal")
	}
	return arg.Value
}

func (t *Transformer) transformStringAssertion(ensure ast.EnsureNode) goast.Expr {
	result := []goast.Expr{}

	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			arg := expectIntLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						transformExpression(ensure.Variable),
					},
				},
				Op: token.LSS,
				Y:  transformExpression(*arg),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			arg := expectIntLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X: &goast.CallExpr{
					Fun: goast.NewIdent("len"),
					Args: []goast.Expr{
						transformExpression(ensure.Variable),
					},
				},
				Op: token.GTR,
				Y:  transformExpression(*arg),
			}
		case "HasPrefix":
			if len(constraint.Args) != 1 {
				panic("HasPrefix constraint requires 1 argument")
			}
			arg := expectStringLiteral(&constraint.Args[0])
			expr = negateCondition(&goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("strings"),
					Sel: goast.NewIdent("HasPrefix"),
				},
				Args: []goast.Expr{
					transformExpression(ensure.Variable),
					transformExpression(*arg),
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
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			arg := expectNumberLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.LSS,
				Y:  transformExpression(*arg),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			arg := expectNumberLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.GTR,
				Y:  transformExpression(*arg),
			}
		case "LessThan":
			if len(constraint.Args) != 1 {
				panic("LessThan constraint requires 1 argument")
			}
			arg := expectNumberLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.GEQ,
				Y:  transformExpression(*arg),
			}
		case "GreaterThan":
			if len(constraint.Args) != 1 {
				panic("GreaterThan constraint requires 1 argument")
			}
			arg := expectNumberLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.LEQ,
				Y:  transformExpression(*arg),
			}
		default:
			panic("Unknown Int constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) transformFloatAssertion(ensure ast.EnsureNode) goast.Expr {
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "Min":
			if len(constraint.Args) != 1 {
				panic("Min constraint requires 1 argument")
			}
			arg := expectNumberLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.LSS,
				Y:  transformExpression(*arg),
			}
		case "Max":
			if len(constraint.Args) != 1 {
				panic("Max constraint requires 1 argument")
			}
			arg := expectNumberLiteral(&constraint.Args[0])
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.GTR,
				Y:  transformExpression(*arg),
			}
		default:
			panic("Unknown Float constraint: " + constraint.Name)
		}
		result = append(result, expr)
	}
	return any(result)
}

func (t *Transformer) transformBoolAssertion(ensure ast.EnsureNode) goast.Expr {
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case "True":
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
				Op: token.EQL,
				Y:  goast.NewIdent(BOOL_CONSTANT_TRUE),
			}
		case "False":
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
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
	result := []goast.Expr{}
	for _, constraint := range ensure.Assertion.Constraints {
		var expr goast.Expr
		switch constraint.Name {
		case NIL_CONSTRAINT:
			expr = &goast.BinaryExpr{
				X:  transformExpression(ensure.Variable),
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

func (t *Transformer) getEnsureBaseType(ensure ast.EnsureNode) ast.TypeNode {
	if ensure.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *ensure.Assertion.BaseType}
	}

	ensureBaseType, err := t.TypeChecker.LookupEnsureBaseType(&ensure, t.currentScope)
	if err != nil {
		panic(err)
	}
	return *ensureBaseType
}

// Helper to look up a TypeGuardNode by name
func (t *Transformer) lookupTypeGuardNode(name string) *ast.TypeGuardNode {
	for _, def := range t.TypeChecker.Defs {
		if tg, ok := def.(*ast.TypeGuardNode); ok {
			if string(tg.Ident) == name {
				return tg
			}
		}
	}
	panic("Type guard not found: " + name)
}

func (t *Transformer) transformEnsureCondition(ensure ast.EnsureNode) goast.Expr {
	// If any constraint name matches a type guard node, treat as a type guard assertion
	for _, constraint := range ensure.Assertion.Constraints {
		for _, def := range t.TypeChecker.Defs {
			if tg, ok := def.(*ast.TypeGuardNode); ok && string(tg.Ident) == constraint.Name {
				return t.transformTypeGuardAssertion(ensure)
			}
		}
	}

	baseType := t.getEnsureBaseType(ensure)

	switch baseType.Ident {
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
		panic("Unknown base type: " + baseType.Ident)
	}
}

func (t *Transformer) transformTypeGuardAssertion(ensure ast.EnsureNode) goast.Expr {
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
					transformExpression(ensure.Variable),
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
			transformExpression(ensure.Variable),
		},
	}
}
