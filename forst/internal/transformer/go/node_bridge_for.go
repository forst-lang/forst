package transformergo

import (
	"fmt"
	"strconv"

	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func (t *Transformer) tryTransformNodeIteratorFor(fn *ast.ForNode) (goast.Stmt, bool, error) {
	info, ok, err := t.nodeIteratorRangeInfo(fn.RangeX)
	if err != nil {
		return nil, true, err
	}
	if !ok {
		return nil, false, nil
	}

	t.appendNodeBridgeIfNeeded()
	if err := t.ensureNodeReturnTypeEmitted(info.elemType); err != nil {
		return nil, true, err
	}
	elemGoType, err := t.transformType(info.elemType)
	if err != nil {
		return nil, true, err
	}
	_, _, stepTypeIdent, err := t.ensureForstNodeSeqTypes(ast.NewResultType(
		ast.TypeNode{Ident: "Seq", TypeParams: []ast.TypeNode{info.elemType}},
		ast.NewBuiltinType(ast.TypeError),
	))
	if err != nil {
		return nil, true, err
	}
	stepGoType := stepTypeIdent
	_ = elemGoType

	itIdent := goast.NewIdent("_nodeIt")
	stepIdent := goast.NewIdent("_nodeStep")
	idxIdent := goast.NewIdent("_nodeIdx")
	batchIdent := goast.NewIdent("_nodeBatch")
	batchIdxIdent := goast.NewIdent("_nodeBatchIdx")
	batchErrIdent := goast.NewIdent("_nodeBatchErr")

	batchType := &goast.ArrayType{Elt: stepGoType}

	openStmt := &goast.AssignStmt{
		Lhs: []goast.Expr{itIdent},
		Tok: token.DEFINE,
		Rhs: []goast.Expr{info.openExpr},
	}

	deferClose := &goast.DeferStmt{
		Call: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   itIdent,
				Sel: goast.NewIdent("Close"),
			},
		},
	}

	body := &goast.BlockStmt{}
	if err := t.emitNodeIteratorLoopBindings(body, fn, stepIdent, idxIdent, elemGoType); err != nil {
		return nil, true, err
	}
	for _, st := range fn.Body {
		gst, err := t.transformStatement(st)
		if err != nil {
			return nil, true, err
		}
		body.List = append(body.List, gst)
	}

	loopBody := &goast.BlockStmt{
		List: []goast.Stmt{
			&goast.IfStmt{
				Cond: &goast.BinaryExpr{
					X:  batchIdxIdent,
					Op: token.GEQ,
					Y: &goast.CallExpr{
						Fun:  goast.NewIdent("len"),
						Args: []goast.Expr{batchIdent},
					},
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.AssignStmt{
							Lhs: []goast.Expr{batchIdent, batchErrIdent},
							Tok: token.ASSIGN,
							Rhs: []goast.Expr{
								&goast.CallExpr{
									Fun: &goast.SelectorExpr{
										X:   itIdent,
										Sel: goast.NewIdent("NextBatch"),
									},
									Args: []goast.Expr{goast.NewIdent(strconv.Itoa(nodeSeqBatchSize))},
								},
							},
						},
						&goast.IfStmt{
							Cond: &goast.BinaryExpr{
								X:  batchErrIdent,
								Op: token.NEQ,
								Y:  goast.NewIdent("nil"),
							},
							Body: &goast.BlockStmt{
								List: []goast.Stmt{&goast.ExprStmt{X: &goast.CallExpr{Fun: goast.NewIdent("panic"), Args: []goast.Expr{batchErrIdent}}}},
							},
						},
						&goast.AssignStmt{
							Lhs: []goast.Expr{batchIdxIdent},
							Tok: token.ASSIGN,
							Rhs: []goast.Expr{goast.NewIdent("0")},
						},
					},
				},
			},
			&goast.AssignStmt{
				Lhs: []goast.Expr{stepIdent},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{
					&goast.IndexExpr{
						X:     batchIdent,
						Index: batchIdxIdent,
					},
				},
			},
			&goast.IncDecStmt{X: batchIdxIdent, Tok: token.INC},
			&goast.IfStmt{
				Cond: &goast.BinaryExpr{
					X: &goast.SelectorExpr{
						X:   stepIdent,
						Sel: goast.NewIdent("Kind"),
					},
					Op: token.EQL,
					Y: goast.NewIdent(forstNodeGenStepDoneIdent),
				},
				Body: &goast.BlockStmt{List: []goast.Stmt{&goast.BranchStmt{Tok: token.BREAK}}},
			},
			&goast.IfStmt{
				Cond: &goast.BinaryExpr{
					X: &goast.SelectorExpr{
						X:   stepIdent,
						Sel: goast.NewIdent("Kind"),
					},
					Op: token.EQL,
					Y: goast.NewIdent(forstNodeGenStepErrorIdent),
				},
				Body: &goast.BlockStmt{
					List: []goast.Stmt{
						&goast.ExprStmt{
							X: &goast.CallExpr{
								Fun: goast.NewIdent("panic"),
								Args: []goast.Expr{
									&goast.SelectorExpr{X: stepIdent, Sel: goast.NewIdent("Message")},
								},
							},
						},
					},
				},
			},
		},
	}
	loopBody.List = append(loopBody.List, body.List...)

	loop := &goast.ForStmt{Body: loopBody}

	prefix := []goast.Stmt{
		openStmt,
		deferClose,
		&goast.DeclStmt{
			Decl: &goast.GenDecl{
				Tok: token.VAR,
				Specs: []goast.Spec{
					&goast.ValueSpec{
						Names: []*goast.Ident{stepIdent},
						Type:  stepGoType,
					},
					&goast.ValueSpec{
						Names: []*goast.Ident{batchIdent},
						Type:  batchType,
					},
					&goast.ValueSpec{
						Names: []*goast.Ident{batchIdxIdent},
						Type:  goast.NewIdent("int"),
					},
					&goast.ValueSpec{
						Names: []*goast.Ident{batchErrIdent},
						Type:  goast.NewIdent("error"),
					},
				},
			},
		},
		&goast.AssignStmt{
			Lhs: []goast.Expr{batchIdent, batchErrIdent},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   itIdent,
						Sel: goast.NewIdent("NextBatch"),
					},
					Args: []goast.Expr{goast.NewIdent(strconv.Itoa(nodeSeqBatchSize))},
				},
			},
		},
		&goast.IfStmt{
			Cond: &goast.BinaryExpr{
				X:  batchErrIdent,
				Op: token.NEQ,
				Y:  goast.NewIdent("nil"),
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{&goast.ExprStmt{X: &goast.CallExpr{Fun: goast.NewIdent("panic"), Args: []goast.Expr{batchErrIdent}}}},
			},
		},
		&goast.AssignStmt{
			Lhs: []goast.Expr{batchIdxIdent},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{goast.NewIdent("0")},
		},
	}
	if fn.RangeKey != nil && fn.RangeValue != nil && fn.RangeKey.ID != "_" {
		prefix = append(prefix, &goast.AssignStmt{
			Lhs: []goast.Expr{idxIdent},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{goast.NewIdent("-1")},
		})
		loopBody.List = append([]goast.Stmt{
			&goast.IncDecStmt{X: idxIdent, Tok: token.INC},
		}, loopBody.List...)
	}

	block := &goast.BlockStmt{
		List: append(prefix, loop),
	}
	if fn.Label != nil {
		return &goast.LabeledStmt{
			Label: goast.NewIdent(string(fn.Label.ID)),
			Stmt:  block,
		}, true, nil
	}
	return block, true, nil
}

type nodeIteratorRangeInfo struct {
	openExpr   goast.Expr
	openInline bool
	elemType   ast.TypeNode
}

func (t *Transformer) nodeIteratorRangeInfo(rangeX ast.ExpressionNode) (nodeIteratorRangeInfo, bool, error) {
	var zero nodeIteratorRangeInfo
	rangeTypes, err := t.TypeChecker.LookupInferredType(rangeX, false)
	if err != nil {
		return zero, false, err
	}
	if len(rangeTypes) != 1 {
		return zero, false, nil
	}
	rt := rangeTypes[0]
	if rt.Ident != "Seq" {
		return zero, false, nil
	}
	elemType, err := nodeSeqElementType(rt)
	if err != nil {
		return zero, true, err
	}

	if call, ok := rangeX.(ast.FunctionCallNode); ok {
		if expr, handled, err := t.transformNodeQualifiedCall(call); handled {
			if err != nil {
				return zero, true, err
			}
			return nodeIteratorRangeInfo{openExpr: expr, openInline: true, elemType: elemType}, true, nil
		}
	}

	expr, err := t.transformExpression(rangeX)
	if err != nil {
		return zero, true, err
	}
	return nodeIteratorRangeInfo{openExpr: expr, openInline: false, elemType: elemType}, true, nil
}

func (t *Transformer) emitNodeIteratorLoopBindings(
	body *goast.BlockStmt,
	fn *ast.ForNode,
	stepIdent *goast.Ident,
	idxIdent *goast.Ident,
	_ goast.Expr,
) error {
	valueFromStep := &goast.SelectorExpr{X: stepIdent, Sel: goast.NewIdent("Value")}
	noKey := fn.RangeKey == nil
	noVal := fn.RangeValue == nil
	if noKey && noVal {
		return nil
	}
	if !noKey && noVal {
		return assignRangeBinding(body, fn.RangeKey, valueFromStep, fn.RangeShort)
	}
	if noKey && !noVal {
		return fmt.Errorf("range loop value variable without key is invalid")
	}
	if fn.RangeKey.ID == "_" {
		return assignRangeBinding(body, fn.RangeValue, valueFromStep, fn.RangeShort)
	}
	if err := assignRangeBinding(body, fn.RangeKey, idxIdent, fn.RangeShort); err != nil {
		return err
	}
	return assignRangeBinding(body, fn.RangeValue, valueFromStep, fn.RangeShort)
}

func assignRangeBinding(body *goast.BlockStmt, id *ast.Ident, rhs goast.Expr, isShort bool) error {
	if id == nil || id.ID == "_" {
		return nil
	}
	name := goast.NewIdent(string(id.ID))
	tok := token.ASSIGN
	if isShort {
		tok = token.DEFINE
	}
	body.List = append(body.List, &goast.AssignStmt{
		Lhs: []goast.Expr{name},
		Tok: tok,
		Rhs: []goast.Expr{rhs},
	})
	return nil
}
