package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
	"strings"

	"github.com/sirupsen/logrus"
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

// transformEnsureConstraints transforms an ensure node based on its type
func (at *AssertionTransformer) transformEnsureConstraints(ensure ast.EnsureNode) (goast.Expr, error) {
	baseType, err := at.transformer.getEnsureBaseType(ensure)
	if err != nil {
		return nil, err
	}

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
		at.transformer.log.Errorf("No type guard found, using assertion identifier: %s", string(ident.Ident))
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

func (t *Transformer) getEnsureBaseType(ensure ast.EnsureNode) (ast.TypeNode, error) {
	if ensure.Assertion.BaseType != nil {
		return ast.TypeNode{Ident: *ensure.Assertion.BaseType}, nil
	}

	ensureBaseType, err := t.TypeChecker.LookupEnsureBaseType(&ensure, t.currentScope())
	if err != nil {
		return ast.TypeNode{}, err
	}
	return *ensureBaseType, nil
}

// Helper to look up a TypeGuardNode by name
func (t *Transformer) lookupTypeGuardNode(name string) *ast.TypeGuardNode {
	for _, def := range t.TypeChecker.Defs {
		if tg, ok := def.(ast.TypeGuardNode); ok {
			t.log.WithFields(logrus.Fields{
				"requested": name,
				"candidate": tg.GetIdent(),
			}).Trace("lookupTypeGuardNode: candidate check")
			if tg.GetIdent() == name {
				t.log.WithFields(logrus.Fields{
					"requested": name,
					"found":     true,
				}).Trace("lookupTypeGuardNode: found match")
				return &tg
			}
		}
	}
	t.log.WithFields(logrus.Fields{
		"requested": name,
		"found":     false,
	}).Trace("lookupTypeGuardNode: not found")
	panic("Type guard not found: " + name)
}

func (t *Transformer) transformEnsureCondition(ensure ast.EnsureNode) (goast.Expr, error) {
	// Look up the variable type first (inferred)
	variableType, err := t.TypeChecker.LookupVariableType(&ensure.Variable, t.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup ensure variable type: %w, scope: %s", err, t.currentScope().String())
	}

	// Look up the declared type from the symbol table
	declaredType := variableType
	parts := strings.Split(string(ensure.Variable.Ident.ID), ".")
	baseIdent := ast.Identifier(parts[0])
	if symbol, exists := t.currentScope().LookupVariable(baseIdent, true); exists {
		if len(symbol.Types) > 0 {
			declaredType = symbol.Types[0]
		}
	}

	t.log.WithFields(logrus.Fields{
		"variable":     ensure.Variable.Ident.ID,
		"declaredType": declaredType.Ident,
		"inferredType": variableType.Ident,
		"assertion":    ensure.Assertion,
	}).Trace("transformEnsureCondition: TypeGuardLookup")

	// --- Type Guard Lookup: Prefer declared type (alias) over base type ---
	var typeGuardExprs []goast.Expr
	// 1. Try declared type (alias) first
	if _, exists := t.TypeChecker.Defs[declaredType.Ident]; exists {
		// Search for type guards defined for this alias
		for _, constraint := range ensure.Assertion.Constraints {
			t.log.Tracef("[transformEnsureCondition] [Alias] Checking constraint: %s, Args: %+v", constraint.Name, constraint.Args)
			for _, def := range t.TypeChecker.Defs {
				if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
					subjectType := tg.Subject.GetType()
					if t.TypeChecker.IsTypeCompatible(declaredType, subjectType) {
						t.log.Tracef("[transformEnsureCondition] [Alias] Type guard '%s' is compatible with declared variable type '%s' (subject type: '%s')", tg.Ident, declaredType.Ident, subjectType.Ident)
						typeGuardExpr := t.transformTypeGuardEnsure(ensure)
						typeGuardExprs = append(typeGuardExprs, typeGuardExpr)
						break // Found a matching type guard for this constraint
					}
				}
			}
		}
	}
	if len(typeGuardExprs) > 0 {
		t.log.Tracef("[transformEnsureCondition] Returning conjoined type guard expressions (alias): %+v", typeGuardExprs)
		return conjoin(typeGuardExprs), nil
	}
	// 2. Fallback: Try base type if no alias type guard found
	baseType, err := t.getEnsureBaseType(ensure)
	if err == nil && baseType.Ident != declaredType.Ident {
		if _, exists := t.TypeChecker.Defs[baseType.Ident]; exists {
			for _, constraint := range ensure.Assertion.Constraints {
				t.log.Tracef("[transformEnsureCondition] [Base] Checking constraint: %s, Args: %+v", constraint.Name, constraint.Args)
				for _, def := range t.TypeChecker.Defs {
					if tg, ok := def.(ast.TypeGuardNode); ok && tg.GetIdent() == constraint.Name {
						subjectType := tg.Subject.GetType()
						if t.TypeChecker.IsTypeCompatible(baseType, subjectType) {
							t.log.Tracef("[transformEnsureCondition] [Base] Type guard '%s' is compatible with base variable type '%s' (subject type: '%s')", tg.Ident, baseType.Ident, subjectType.Ident)
							typeGuardExpr := t.transformTypeGuardEnsure(ensure)
							typeGuardExprs = append(typeGuardExprs, typeGuardExpr)
							break // Found a matching type guard for this constraint
						}
					}
				}
			}
		}
	}
	if len(typeGuardExprs) > 0 {
		t.log.Tracef("[transformEnsureCondition] Returning conjoined type guard expressions (base): %+v", typeGuardExprs)
		return conjoin(typeGuardExprs), nil
	}

	t.log.Tracef("No type guard found, transforming ensure condition, var type %+v, assertion: %+v", variableType, ensure.Assertion)
	expr, err := t.assertionTransformer.transformEnsureConstraints(ensure)
	if err != nil {
		return nil, fmt.Errorf("failed to transform ensure conditions constraints: %w", err)
	}
	return expr, nil
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

func (t *AssertionTransformer) transformEnsureCondition(ensure *ast.EnsureNode) ([]goast.Stmt, error) {
	// Get the variable type from the symbol table
	varType, err := t.transformer.TypeChecker.LookupVariableType(&ensure.Variable, t.transformer.currentScope())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup variable type: %w", err)
	}

	// Log variable type information
	t.transformer.log.WithFields(logrus.Fields{
		"variable":     ensure.Variable.GetIdent(),
		"declaredType": varType.Ident,
		"inferredType": t.transformer.TypeChecker.InferredTypes[t.transformer.TypeChecker.Hasher.HashNode(ensure.Variable)][0].Ident,
	}).Trace("[transformEnsureCondition] Variable type information")

	// Transform each constraint
	var transformedStmts []goast.Stmt
	for _, constraint := range ensure.Assertion.Constraints {
		// Log constraint details
		t.transformer.log.WithFields(logrus.Fields{
			"constraint":     constraint.String(),
			"constraintType": fmt.Sprintf("%T", constraint),
		}).Trace("[transformEnsureCondition] Processing constraint")

		// Check if this constraint name matches a registered type guard
		typeGuardDef := t.transformer.lookupTypeGuardNode(constraint.Name)
		if typeGuardDef != nil {
			t.transformer.log.WithFields(logrus.Fields{
				"typeGuard":     constraint.Name,
				"typeGuardDef":  typeGuardDef,
				"typeGuardType": fmt.Sprintf("%T", typeGuardDef),
			}).Trace("[transformEnsureCondition] Found type guard constraint by name")

			// Check if the type guard is compatible with the variable type
			if t.isTypeGuardCompatible(varType, typeGuardDef) {
				t.transformer.log.WithFields(logrus.Fields{
					"typeGuard": constraint.Name,
					"varType":   varType.Ident,
				}).Trace("[transformEnsureCondition] Type guard is compatible")

				// Transform the type guard into a function call
				transformed := t.transformer.transformTypeGuardEnsure(ast.EnsureNode{
					Variable:  ensure.Variable,
					Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{constraint}},
				})
				transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
				continue
			}

			t.transformer.log.WithFields(logrus.Fields{
				"typeGuard": constraint.Name,
				"varType":   varType.Ident,
			}).Trace("[transformEnsureCondition] Type guard is not compatible")
		}

		// If not a type guard or not compatible, try to transform as a regular constraint
		t.transformer.log.WithFields(logrus.Fields{
			"constraint": constraint.String(),
		}).Trace("[transformEnsureCondition] Attempting to transform as regular constraint")

		transformed, err := t.transformer.assertionTransformer.transformEnsureConstraints(ast.EnsureNode{
			Variable:  ensure.Variable,
			Assertion: ast.AssertionNode{Constraints: []ast.ConstraintNode{constraint}},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to transform ensure conditions constraints: %w", err)
		}
		transformedStmts = append(transformedStmts, &goast.ExprStmt{X: transformed})
	}

	return transformedStmts, nil
}

func (t *AssertionTransformer) isTypeGuardCompatible(varType ast.TypeNode, typeGuard *ast.TypeGuardNode) bool {
	t.transformer.log.WithFields(logrus.Fields{
		"varType":   varType.Ident,
		"typeGuard": typeGuard.GetIdent(),
	}).Trace("[isTypeGuardCompatible] Checking type guard compatibility")

	// Get the base type of the variable
	baseType, err := t.transformer.getEnsureBaseType(ast.EnsureNode{Variable: ast.VariableNode{ExplicitType: varType}})
	if err != nil {
		t.transformer.log.WithError(err).Error("[isTypeGuardCompatible] Failed to get base type")
		return false
	}
	t.transformer.log.WithFields(logrus.Fields{
		"baseType": baseType.Ident,
	}).Trace("[isTypeGuardCompatible] Base type")

	// Check if the type guard is defined for the base type
	for _, param := range typeGuard.Parameters() {
		t.transformer.log.WithFields(logrus.Fields{
			"paramType": param.GetType().Ident,
			"baseType":  baseType.Ident,
		}).Trace("[isTypeGuardCompatible] Checking parameter type")

		if param.GetType().Ident == baseType.Ident {
			t.transformer.log.WithFields(logrus.Fields{
				"typeGuard": typeGuard.GetIdent(),
				"baseType":  baseType.Ident,
			}).Trace("[isTypeGuardCompatible] Found compatible type guard")
			return true
		}
	}

	t.transformer.log.WithFields(logrus.Fields{
		"typeGuard": typeGuard.GetIdent(),
		"baseType":  baseType.Ident,
	}).Trace("[isTypeGuardCompatible] No compatible type guard found")
	return false
}
