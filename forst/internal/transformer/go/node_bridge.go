package transformergo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	goasttoken "go/token"
)

const nodeSeqBatchSize = 32

func (t *Transformer) transformNodeQualifiedCall(e ast.FunctionCallNode) (goast.Expr, bool, error) {
	if t == nil || t.TypeChecker == nil {
		return nil, false, nil
	}
	parts := strings.Split(string(e.Function.ID), ".")
	if len(parts) != 2 {
		return nil, false, nil
	}
	target, ok := t.TypeChecker.NodeCallTarget(parts[0], parts[1])
	if !ok {
		return nil, false, nil
	}

	retTypes, err := t.TypeChecker.LookupInferredType(e, false)
	if err != nil || len(retTypes) != 1 {
		return nil, true, fmt.Errorf("codegen: node call %s: missing inferred return type", e.Function.ID)
	}

	switch target.Kind {
	case typechecker.NodeExportKindFunction:
		return t.transformNodeSyncCall(e, target, retTypes[0])
	case typechecker.NodeExportKindAsyncFunction:
		return t.transformNodeAsyncCall(e, target, retTypes[0])
	case typechecker.NodeExportKindGenerator, typechecker.NodeExportKindAsyncGenerator:
		return t.transformNodeGenOpen(e, target, retTypes[0], target.Kind == typechecker.NodeExportKindAsyncGenerator)
	default:
		return nil, true, fmt.Errorf("codegen: node export %s kind %q is not yet lowered to Go", target.ExportName, target.Kind)
	}
}

func (t *Transformer) transformNodeSyncCall(e ast.FunctionCallNode, target typechecker.NodeCallTarget, ret ast.TypeNode) (goast.Expr, bool, error) {
	return t.transformNodeBridgeCall(e, target, ret, "CallSync")
}

func (t *Transformer) transformNodeAsyncCall(e ast.FunctionCallNode, target typechecker.NodeCallTarget, ret ast.TypeNode) (goast.Expr, bool, error) {
	return t.transformNodeBridgeCall(e, target, ret, "CallAsync")
}

func (t *Transformer) transformNodeBridgeCall(e ast.FunctionCallNode, target typechecker.NodeCallTarget, ret ast.TypeNode, bridgeFn string) (goast.Expr, bool, error) {
	valueType, err := nodeBridgeCallValueType(ret)
	if err != nil {
		return nil, true, err
	}
	if valueType.Ident != ast.TypeVoid {
		if err := t.ensureNodeReturnTypeEmitted(valueType); err != nil {
			return nil, true, err
		}
	}
	wrapperName, callArgs, err := t.registerNodeCallWrapper(target, ret, bridgeFn, e)
	if err != nil {
		return nil, true, err
	}
	return &goast.CallExpr{
		Fun:  goast.NewIdent(wrapperName),
		Args: callArgs,
	}, true, nil
}

// nodeBridgeCallValueType returns the success payload type for nodert.Call* generics.
// Node qualified calls are typed as Result(T, Error) in Forst; Go CallSync/CallAsync return (T, error).
func nodeBridgeCallValueType(ret ast.TypeNode) (ast.TypeNode, error) {
	if ret.IsResultType() && len(ret.TypeParams) >= 1 {
		return ret.TypeParams[0], nil
	}
	return ret, nil
}

func (t *Transformer) nodeBridgeResultGoType(ret ast.TypeNode) (goast.Expr, error) {
	if ret.Ident == ast.TypeVoid {
		return &goast.StructType{Fields: &goast.FieldList{}}, nil
	}
	return t.transformType(ret)
}

func (t *Transformer) transformNodeGenOpen(e ast.FunctionCallNode, target typechecker.NodeCallTarget, ret ast.TypeNode, async bool) (goast.Expr, bool, error) {
	seqType, err := nodeBridgeCallValueType(ret)
	if err != nil {
		return nil, true, err
	}
	elemType, err := nodeSeqElementType(seqType)
	if err != nil {
		return nil, true, err
	}
	if err := t.ensureNodeReturnTypeEmitted(elemType); err != nil {
		return nil, true, err
	}
	wrapperName, callArgs, err := t.registerNodeOpenSeqWrapper(target, ret, e, async)
	if err != nil {
		return nil, true, err
	}
	return &goast.CallExpr{
		Fun:  goast.NewIdent(wrapperName),
		Args: callArgs,
	}, true, nil
}

func nodeSeqElementType(ret ast.TypeNode) (ast.TypeNode, error) {
	if ret.Ident == "Seq" && len(ret.TypeParams) >= 1 {
		return ret.TypeParams[0], nil
	}
	return ast.TypeNode{}, fmt.Errorf("codegen: expected Seq return type, got %s", ret.Ident)
}

func (t *Transformer) ensureNodeReturnTypeEmitted(ret ast.TypeNode) error {
	if t == nil || t.TypeChecker == nil {
		return nil
	}
	if ret.Assertion == nil {
		return t.ensureTypeEmittedFromForstType(ret)
	}
	resolved, err := t.TypeChecker.LookupAssertionType(ret.Assertion)
	if err != nil {
		return err
	}
	typeIdent := resolved.Ident
	processed := make(map[ast.TypeIdent]bool)
	if shape := shapeFromAssertion(ret.Assertion); shape != nil {
		return t.emitTypeAndReferencedTypes(typeIdent, ast.TypeDefNode{
			Ident: typeIdent,
			Expr:  ast.TypeDefShapeExpr{Shape: *shape},
		}, processed)
	}
	if _, exists := t.TypeChecker.Defs[typeIdent]; !exists {
		t.TypeChecker.Defs[typeIdent] = ast.TypeDefNode{
			Ident: typeIdent,
			Expr: ast.TypeDefAssertionExpr{
				Assertion: ret.Assertion,
			},
		}
	}
	return t.emitTypeAndReferencedTypes(typeIdent, t.TypeChecker.Defs[typeIdent], processed)
}

func (t *Transformer) ensureTypeEmittedFromForstType(ft ast.TypeNode) error {
	if t == nil || t.TypeChecker == nil || t.Output == nil {
		return nil
	}
	if ft.Ident == "" || ft.Ident == ast.TypeImplicit {
		return nil
	}
	if t.Output.HasType(string(ft.Ident)) {
		return nil
	}
	processed := make(map[ast.TypeIdent]bool)
	if def, ok := t.TypeChecker.Defs[ft.Ident]; ok {
		return t.emitTypeAndReferencedTypes(ft.Ident, def, processed)
	}
	if shape := shapeFromAssertion(ft.Assertion); shape != nil {
		minimalDef := ast.TypeDefNode{
			Ident: ft.Ident,
			Expr:  ast.TypeDefShapeExpr{Shape: *shape},
		}
		return t.emitTypeAndReferencedTypes(ft.Ident, minimalDef, processed)
	}
	return nil
}

func shapeFromAssertion(asn *ast.AssertionNode) *ast.ShapeNode {
	if asn == nil {
		return nil
	}
	for _, c := range asn.Constraints {
		for _, arg := range c.Args {
			if arg.Shape != nil {
				cp := *arg.Shape
				return &cp
			}
		}
	}
	return nil
}

func nodeBridgeStringLit(s string) *goast.BasicLit {
	return &goast.BasicLit{Kind: goasttoken.STRING, Value: strconv.Quote(s)}
}

// tryEmitStaticNodeCallArgsJSON returns a json.RawMessage literal when every arg is a compile-time literal.
func (t *Transformer) tryEmitStaticNodeCallArgsJSON(args []ast.ExpressionNode) (goast.Expr, bool) {
	if len(args) == 0 {
		return &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("json"),
				Sel: goast.NewIdent("RawMessage"),
			},
			Args: []goast.Expr{nodeBridgeStringLit("[]")},
		}, true
	}
	values := make([]any, 0, len(args))
	for _, arg := range args {
		v, ok := staticNodeCallArgValue(arg)
		if !ok {
			return nil, false
		}
		values = append(values, v)
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, false
	}
	return &goast.CallExpr{
		Fun: &goast.SelectorExpr{
			X:   goast.NewIdent("json"),
			Sel: goast.NewIdent("RawMessage"),
		},
		Args: []goast.Expr{nodeBridgeStringLit(string(raw))},
	}, true
}

func staticNodeCallArgValue(arg ast.ExpressionNode) (any, bool) {
	switch a := arg.(type) {
	case ast.IntLiteralNode:
		return a.Value, true
	case ast.FloatLiteralNode:
		return a.Value, true
	case ast.StringLiteralNode:
		return a.Value, true
	case ast.BoolLiteralNode:
		return a.Value, true
	default:
		return nil, false
	}
}
