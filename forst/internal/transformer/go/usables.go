package transformergo

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/hasher"
	"forst/internal/typechecker"
	goast "go/ast"
	goasttoken "go/token"
	"sort"
	"strings"
)

// wiringFrame holds merged Usable wiring field sources for with-block lowering.
type wiringFrame struct {
	fields   map[string]goast.Expr
	baseExpr goast.Expr
}

func usablesSlotSetKey(slots []typechecker.UsableSlot) string {
	return strings.Join(rootIdentsFromSlots(slots), ",")
}

func rootIdentsFromSlots(slots []typechecker.UsableSlot) []string {
	roots := make([]string, len(slots))
	for i, s := range slots {
		roots[i] = string(s.RootIdent)
	}
	sort.Strings(roots)
	return roots
}

func (t *Transformer) usablesStructName(slots []typechecker.UsableSlot) string {
	key := usablesSlotSetKey(slots)
	if name, ok := t.usablesStructByKey[key]; ok {
		return name
	}
	name := hasher.HashSortedStrings(rootIdentsFromSlots(slots)...).ToUsablesIdent()
	t.usablesStructByKey[key] = name
	return name
}

func (t *Transformer) emitAllUsablesStructs() error {
	seen := make(map[string]struct{})
	for _, slots := range t.TypeChecker.FunctionUsables {
		if len(slots) == 0 {
			continue
		}
		key := usablesSlotSetKey(slots)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if err := t.emitUsablesStruct(slots); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transformer) emitUsablesStruct(slots []typechecker.UsableSlot) error {
	name := t.usablesStructName(slots)
	if t.Output.HasType(name) {
		return nil
	}
	fields := make([]*goast.Field, 0, len(slots))
	for _, slot := range slots {
		fieldType, err := t.transformType(slot.ContractType)
		if err != nil {
			return fmt.Errorf("usables struct field %s: %w", slot.RootIdent, err)
		}
		fields = append(fields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(string(slot.RootIdent))},
			Type:  fieldType,
		})
	}
	t.Output.AddType(&goast.GenDecl{
		Tok: goasttoken.TYPE,
		Specs: []goast.Spec{
			&goast.TypeSpec{
				Name: goast.NewIdent(name),
				Type: &goast.StructType{
					Fields: &goast.FieldList{List: fields},
				},
			},
		},
	})
	return nil
}

func (t *Transformer) isUsablesWiringRoot(fn ast.FunctionNode) bool {
	return ast.IsUsablesWiringRoot(fn.Ident.ID, ast.ParamTypesFromFunction(fn))
}

func (t *Transformer) functionNeedsUsablesParam(fn ast.FunctionNode) bool {
	if fn.Receiver != nil {
		return false
	}
	if t.isUsablesWiringRoot(fn) {
		return false
	}
	return len(t.TypeChecker.FunctionUsables[fn.Ident.ID]) > 0
}

func (t *Transformer) prependUsablesParam(params *goast.FieldList, fn ast.FunctionNode) (*goast.FieldList, error) {
	slots := t.TypeChecker.FunctionUsables[fn.Ident.ID]
	if len(slots) == 0 || fn.Receiver != nil || t.isUsablesWiringRoot(fn) {
		return params, nil
	}
	if err := t.emitUsablesStruct(slots); err != nil {
		return nil, err
	}
	usablesField := &goast.Field{
		Names: []*goast.Ident{goast.NewIdent("usables")},
		Type:  goast.NewIdent(t.usablesStructName(slots)),
	}
	if params == nil {
		return &goast.FieldList{List: []*goast.Field{usablesField}}, nil
	}
	out := &goast.FieldList{List: make([]*goast.Field, 0, len(params.List)+1)}
	out.List = append(out.List, usablesField)
	out.List = append(out.List, params.List...)
	return out, nil
}

func (t *Transformer) transformUseStatement(use ast.UseNode) (goast.Stmt, error) {
	if use.Ident == nil {
		return &goast.EmptyStmt{}, nil
	}
	root := string(t.TypeChecker.UsableRootIdent(use.ContractType))
	return &goast.AssignStmt{
		Lhs: []goast.Expr{goast.NewIdent(string(use.Ident.ID))},
		Tok: goasttoken.DEFINE,
		Rhs: []goast.Expr{
			&goast.SelectorExpr{
				X:   goast.NewIdent("usables"),
				Sel: goast.NewIdent(root),
			},
		},
	}, nil
}

func (t *Transformer) transformWiringFieldExpr(expr ast.ExpressionNode) (goast.Expr, error) {
	if ref, ok := expr.(ast.ReferenceNode); ok {
		inner, err := t.transformExpression(ref.Value)
		if err != nil {
			return nil, err
		}
		return &goast.UnaryExpr{Op: goasttoken.AND, X: inner}, nil
	}
	return t.transformExpression(expr)
}

func (t *Transformer) buildWiringFrame(wiring ast.ExpressionNode) (wiringFrame, error) {
	switch w := wiring.(type) {
	case ast.ShapeNode:
		fields := make(map[string]goast.Expr)
		for fieldName, field := range w.Fields {
			if field.IsMethod {
				continue
			}
			expr, ok := field.ValueExpression()
			if !ok {
				continue
			}
			goExpr, err := t.transformWiringFieldExpr(expr)
			if err != nil {
				return wiringFrame{}, fmt.Errorf("wiring field %s: %w", fieldName, err)
			}
			fields[fieldName] = goExpr
		}
		return wiringFrame{fields: fields}, nil
	default:
		base, err := t.transformExpression(wiring)
		if err != nil {
			return wiringFrame{}, err
		}
		return wiringFrame{baseExpr: base}, nil
	}
}

func mergeWiringFrames(outer, inner wiringFrame) wiringFrame {
	merged := wiringFrame{
		fields: make(map[string]goast.Expr),
	}
	for k, v := range outer.fields {
		merged.fields[k] = v
	}
	for k, v := range inner.fields {
		merged.fields[k] = v
	}
	if inner.baseExpr != nil {
		merged.baseExpr = inner.baseExpr
	} else {
		merged.baseExpr = outer.baseExpr
	}
	return merged
}

func (t *Transformer) pushWiringFrame(with ast.WithNode) error {
	inner, err := t.buildWiringFrame(with.Wiring)
	if err != nil {
		return err
	}
	var merged wiringFrame
	if len(t.wiringStack) > 0 {
		merged = mergeWiringFrames(t.wiringStack[len(t.wiringStack)-1], inner)
	} else {
		merged = inner
	}
	t.wiringStack = append(t.wiringStack, merged)
	return nil
}

func (t *Transformer) popWiringFrame() {
	if len(t.wiringStack) > 0 {
		t.wiringStack = t.wiringStack[:len(t.wiringStack)-1]
	}
}

func (t *Transformer) wiringFieldExpr(rootIdent string) goast.Expr {
	for i := len(t.wiringStack) - 1; i >= 0; i-- {
		if ex, ok := t.wiringStack[i].fields[rootIdent]; ok {
			return ex
		}
	}
	for i := len(t.wiringStack) - 1; i >= 0; i-- {
		if t.wiringStack[i].baseExpr != nil {
			return &goast.SelectorExpr{
				X:   t.wiringStack[i].baseExpr,
				Sel: goast.NewIdent(rootIdent),
			}
		}
	}
	return nil
}

func (t *Transformer) buildUsablesStructLiteral(slots []typechecker.UsableSlot) (goast.Expr, error) {
	if len(slots) == 0 {
		return nil, fmt.Errorf("buildUsablesStructLiteral: empty slots")
	}
	if err := t.emitUsablesStruct(slots); err != nil {
		return nil, err
	}
	structName := t.usablesStructName(slots)

	if len(t.wiringStack) == 0 {
		if t.currentFnUsablesName != "" && usablesSlotSetKey(t.currentFnUsablesSlots) == usablesSlotSetKey(slots) {
			return goast.NewIdent(t.currentFnUsablesName), nil
		}
		elts := make([]goast.Expr, 0, len(slots))
		for _, slot := range slots {
			root := string(slot.RootIdent)
			elts = append(elts, &goast.KeyValueExpr{
				Key: goast.NewIdent(root),
				Value: &goast.SelectorExpr{
					X:   goast.NewIdent(t.currentFnUsablesName),
					Sel: goast.NewIdent(root),
				},
			})
		}
		return &goast.CompositeLit{
			Type: goast.NewIdent(structName),
			Elts: elts,
		}, nil
	}

	elts := make([]goast.Expr, 0, len(slots))
	for _, slot := range slots {
		root := string(slot.RootIdent)
		fieldExpr := t.wiringFieldExpr(root)
		if fieldExpr == nil {
			return nil, fmt.Errorf("wiring field %s not available", root)
		}
		elts = append(elts, &goast.KeyValueExpr{
			Key:   goast.NewIdent(root),
			Value: fieldExpr,
		})
	}
	return &goast.CompositeLit{
		Type: goast.NewIdent(structName),
		Elts: elts,
	}, nil
}

func (t *Transformer) transformFunctionCallArgs(callee ast.Identifier, args []ast.ExpressionNode) ([]goast.Expr, error) {
	slots := t.TypeChecker.FunctionUsables[callee]
	paramTypes := make([]ast.TypeNode, len(args))
	if sig, ok := t.TypeChecker.Functions[callee]; ok && len(sig.Parameters) == len(args) {
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
	goArgs := make([]goast.Expr, len(args))
	for i, arg := range args {
		if shapeArg, ok := arg.(ast.ShapeNode); ok && paramTypes[i].Ident != ast.TypeImplicit {
			context := &ShapeContext{
				ExpectedType:   &paramTypes[i],
				FunctionName:   string(callee),
				ParameterIndex: i,
			}
			expectedTypeForShape := t.getExpectedTypeForShape(&shapeArg, context)
			argExpr, err := t.transformShapeNodeWithExpectedType(&shapeArg, expectedTypeForShape)
			if err != nil {
				return nil, err
			}
			goArgs[i] = argExpr
		} else {
			argExpr, err := t.transformExpression(arg)
			if err != nil {
				return nil, err
			}
			goArgs[i] = argExpr
		}
	}
	if len(slots) > 0 {
		usablesLit, err := t.buildUsablesStructLiteral(slots)
		if err != nil {
			return nil, err
		}
		goArgs = append([]goast.Expr{usablesLit}, goArgs...)
	}
	return goArgs, nil
}

func (t *Transformer) transformWithStatements(with ast.WithNode) ([]goast.Stmt, error) {
	if err := t.pushWiringFrame(with); err != nil {
		return nil, err
	}
	defer t.popWiringFrame()

	stmts := make([]goast.Stmt, 0, len(with.Body))
	for _, stmt := range with.Body {
		if err := t.restoreScopeForWith(with); err != nil {
			return nil, err
		}
		if withStmt, ok := stmt.(ast.WithNode); ok {
			nested, err := t.transformWithStatements(withStmt)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, nested...)
			continue
		}
		goStmt, err := t.transformStatement(stmt)
		if err != nil {
			return nil, fmt.Errorf("with body: %w", err)
		}
		if _, ok := goStmt.(*goast.EmptyStmt); ok {
			if use, ok := stmt.(ast.UseNode); ok && use.Ident == nil {
				continue
			}
		}
		stmts = append(stmts, goStmt)
	}
	return stmts, nil
}

func (t *Transformer) restoreScopeForWith(with ast.WithNode) error {
	return t.TypeChecker.RestoreScope(with)
}
