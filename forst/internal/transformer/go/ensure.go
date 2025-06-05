package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"

	log "github.com/sirupsen/logrus"
)

const (
	// MinConstraint is the built-in Min constraint in Forst
	MinConstraint = "Min"
	// MaxConstraint is the built-in Max constraint in Forst
	MaxConstraint = "Max"
	// LessThanConstraint is the built-in LessThan constraint in Forst
	LessThanConstraint = "LessThan"
	// GreaterThanConstraint is the built-in GreaterThan constraint in Forst
	GreaterThanConstraint = "GreaterThan"
	// HasPrefixConstraint is the built-in HasPrefix constraint in Forst
	HasPrefixConstraint = "HasPrefix"
	// TrueConstraint is the built-in True constraint in Forst
	TrueConstraint = "True"
	// FalseConstraint is the built-in False constraint in Forst
	FalseConstraint = "False"
	// NilConstraint is the built-in Nil constraint in Forst
	NilConstraint = "Nil"
)

const (
	// BoolConstantTrue is the true constant in Go
	BoolConstantTrue = "true"
	// BoolConstantFalse is the false constant in Go
	BoolConstantFalse = "false"
	// NilConstant is the nil constant in Go
	NilConstant = "nil"
)

// expectValue validates and returns a value node
func (at *AssertionTransformer) expectValue(arg *ast.ConstraintArgumentNode) (ast.ValueNode, error) {
	if arg == nil {
		return nil, fmt.Errorf("expected an argument")
	}

	if arg.Value == nil {
		return nil, fmt.Errorf("expected argument to be a value")
	}

	return *arg.Value, nil
}

// AssertionTransformer handles the transformation of assertions
type AssertionTransformer struct {
	transformer *Transformer
}

// NewAssertionTransformer creates a new AssertionTransformer
func NewAssertionTransformer(t *Transformer) *AssertionTransformer {
	return &AssertionTransformer{transformer: t}
}

// transformEnsure transforms an ensure node based on its type
func (at *AssertionTransformer) transformEnsure(ensure ast.EnsureNode) (goast.Expr, error) {
	baseType := at.transformer.getEnsureBaseType(ensure)

	switch baseType.Ident {
	case ast.TypeString:
		return at.transformStringEnsure(ensure)
	case ast.TypeInt:
		return at.transformIntEnsure(ensure)
	case ast.TypeFloat:
		return at.transformFloatEnsure(ensure)
	case ast.TypeBool:
		return at.transformBoolEnsure(ensure)
	case ast.TypeError:
		return at.transformErrorEnsure(ensure)
	default:
		ident, err := at.transformer.TypeChecker.LookupAssertionType(&ensure.Assertion)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup assertion type: %s", err)
		}
		log.Errorf("No type guard found, using assertion identifier: %s", string(ident.Ident))
		return goast.NewIdent(string(ident.Ident)), nil
	}
}

// validateConstraintArgs validates the number of arguments for a constraint
func (at *AssertionTransformer) validateConstraintArgs(constraint ast.ConstraintNode, expectedArgs int) error {
	if len(constraint.Args) != expectedArgs {
		return fmt.Errorf("%s constraint requires %d argument(s)", constraint.Name, expectedArgs)
	}
	return nil
}

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
		if tg, ok := def.(ast.TypeGuardNode); ok {
			if tg.GetIdent() == name {
				return &tg
			}
		}
	}
	panic("Type guard not found: " + name)
}

func (t *Transformer) transformEnsureCondition(ensure ast.EnsureNode) goast.Expr {
	// If any constraint name matches a type guard node, treat as a type guard assertion
	// Look up the variable type first
	variableType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope)
	if err != nil {
		panic(fmt.Errorf("failed to lookup ensure variable type: %w", err))
	}

	var typeGuardExprs []goast.Expr
	for _, constraint := range ensure.Assertion.Constraints {
		for _, def := range t.TypeChecker.Defs {
			log.Tracef("Checking def: %+v, %s, %s", def, def.Kind(), def.String())

			if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
				log.Tracef("Found type guard node: %s", tg.Ident)
				// TODO: Also validate that the variable is a subtype of the type guard's subject type
				typeGuardExprs = append(typeGuardExprs, t.transformTypeGuardEnsure(ensure))
			}
		}
	}

	if len(typeGuardExprs) > 0 {
		log.Tracef("Type guard found, transformed all constraints into type guard expressions: %+v", typeGuardExprs)
		return conjoin(typeGuardExprs)
	}

	// Variable could be a subtype of a built-in type, so we need to check that as well
	if len(ensure.Assertion.Constraints) > 0 {
		log.Tracef("No type guard found, transforming ensure condition, var type: %+v", variableType)

		baseType := t.getEnsureBaseType(ensure)
		// Look up the type definition for the base type
		if bt, exists := t.TypeChecker.Defs[baseType.Ident]; exists {
			log.Tracef("Found type definition for base type %s: %+v", baseType.Ident, bt)
			if typeDef, ok := bt.(ast.TypeDefNode); ok {
				log.Tracef("Found type definition for base type %s: %+v", baseType.Ident, typeDef)
				if typeDefExpr, ok := typeDef.Expr.(*ast.TypeDefAssertionExpr); ok {
					log.Tracef("Transforming super type ensure condition for base type %s: %+v", baseType.Ident, typeDefExpr)
					superTypeEnsureNode := ast.EnsureNode{
						Variable: ensure.Variable,
						Assertion: ast.AssertionNode{
							BaseType:    (*typeDefExpr).Assertion.BaseType,
							Constraints: ensure.Assertion.Constraints,
						},
					}
					condition := t.transformEnsureCondition(superTypeEnsureNode)
					return condition
				}
			}
		}
		log.Tracef("Base type: %+v", baseType)
	}

	log.Tracef("No type guard found, transforming ensure condition, var type %+v, assertion: %+v", variableType, ensure.Assertion)
	expr, err := t.assertionTransformer.transformEnsure(ensure)
	if err != nil {
		panic(err)
	}
	return expr
}

func (t *Transformer) transformTypeGuardEnsure(ensure ast.EnsureNode) goast.Expr {
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
					t.transformExpression(ensure.Variable),
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
			t.transformExpression(ensure.Variable),
		},
	}
}
