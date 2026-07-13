package transformergo

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"forst/internal/ast"
	"forst/internal/typechecker"
	goast "go/ast"
	goasttoken "go/token"
)

const (
	forstNodeRuntimeFileStem   = "forst_node_runtime.gen"
	forstNodeGenStepDoneIdent  = "forstNodeGenStepDone"
	forstNodeGenStepErrorIdent = "forstNodeGenStepError"
	forstNodeGenStepTypePrefix = "forstNodeGenStep"
	forstNodeSeqTypePrefix     = "forstNodeSeq"
	nodertImportPath           = "forst/nodert"
)

// ForstNodeRuntimeFileName is the base name (without .go) for the generated node runtime companion file.
func ForstNodeRuntimeFileName() string {
	return forstNodeRuntimeFileStem
}

var nodeWrapperSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func (t *Transformer) nodeRuntimeOut() *TransformerOutput {
	if t == nil {
		return nil
	}
	if t.NodeRuntimeOutput == nil {
		t.NodeRuntimeOutput = &TransformerOutput{}
		t.NodeRuntimeOutput.SetPackageName(t.Output.PackageName())
	}
	return t.NodeRuntimeOutput
}

func (t *Transformer) appendNodeBridgeIfNeeded() {
	if t == nil || t.TypeChecker == nil || !EmitNeedsNodeRuntime(t.TypeChecker) {
		return
	}
	if t.nodeRuntimeOut() == nil {
		return
	}
	t.ensureNodeRuntimeScaffold()
	t.appendNodeManifestToRuntime()
	t.appendNodeRuntimeInitIfNeeded()
	t.appendNodeHostShutdownHelperIfNeeded()
}

func (t *Transformer) appendNodeHostShutdownHelperIfNeeded() {
	out := t.nodeRuntimeOut()
	if out == nil || !t.EmbedNodeHostMode || out.HasFunction("ForstNodeWaitForShutdown") {
		return
	}
	t.ensureNodertImportInRuntime()
	out.AddFunction(&goast.FuncDecl{
		Name: goast.NewIdent("ForstNodeWaitForShutdown"),
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ExprStmt{X: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   goast.NewIdent("nodert"),
					Sel: goast.NewIdent("WaitForShutdown"),
				},
			}},
		}},
	})
}

func (t *Transformer) ensureNodeRuntimeScaffold() {
	out := t.nodeRuntimeOut()
	if out == nil || out.HasValueDecl(forstNodeGenStepDoneIdent) {
		return
	}
	out.AddValueDecl(&goast.GenDecl{
		Tok: goasttoken.CONST,
		Specs: []goast.Spec{
			&goast.ValueSpec{
				Names:  []*goast.Ident{goast.NewIdent(forstNodeGenStepDoneIdent)},
				Values: []goast.Expr{nodeBridgeStringLit("done")},
			},
			&goast.ValueSpec{
				Names:  []*goast.Ident{goast.NewIdent(forstNodeGenStepErrorIdent)},
				Values: []goast.Expr{nodeBridgeStringLit("error")},
			},
		},
	})
}

func (t *Transformer) appendNodeManifestToRuntime() {
	if t == nil || t.TypeChecker == nil || !EmitNeedsNodeRuntime(t.TypeChecker) {
		return
	}
	AppendNodeManifestDecl(t.nodeRuntimeOut(), t.TypeChecker.NodeRuntimeInfo().ManifestJSON)
}

func (t *Transformer) appendNodeRuntimeInitIfNeeded() {
	out := t.nodeRuntimeOut()
	if out == nil || !EmitNeedsNodeRuntime(t.TypeChecker) {
		return
	}
	if out.HasFunction("init") {
		return
	}
	t.ensureNodertImportInRuntime()
	out.AddFunction(&goast.FuncDecl{
		Name: goast.NewIdent("init"),
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.ExprStmt{
					X: &goast.CallExpr{
						Fun: &goast.SelectorExpr{
							X:   goast.NewIdent("nodert"),
							Sel: goast.NewIdent("MustConfigureFromManifest"),
						},
						Args: []goast.Expr{goast.NewIdent(forstNodeManifestVarName)},
					},
				},
			},
		},
	})
}

func (t *Transformer) ensureNodertImportInRuntime() {
	out := t.nodeRuntimeOut()
	if out == nil {
		return
	}
	for _, imp := range out.imports {
		if len(imp.Specs) == 0 {
			continue
		}
		spec, ok := imp.Specs[0].(*goast.ImportSpec)
		if !ok || spec.Path == nil {
			continue
		}
		if strings.Trim(spec.Path.Value, `"`) == nodertImportPath {
			return
		}
	}
	out.AddImport(&goast.GenDecl{
		Tok: goasttoken.IMPORT,
		Specs: []goast.Spec{
			&goast.ImportSpec{
				Path: &goast.BasicLit{
					Kind:  goasttoken.STRING,
					Value: strconv.Quote(nodertImportPath),
				},
			},
		},
	})
}

func (t *Transformer) ensureJSONImportInRuntime() {
	out := t.nodeRuntimeOut()
	if out == nil {
		return
	}
	for _, imp := range out.imports {
		if len(imp.Specs) == 0 {
			continue
		}
		spec, ok := imp.Specs[0].(*goast.ImportSpec)
		if !ok || spec.Path == nil {
			continue
		}
		if strings.Trim(spec.Path.Value, `"`) == "encoding/json" {
			return
		}
	}
	out.AddImport(&goast.GenDecl{
		Tok: goasttoken.IMPORT,
		Specs: []goast.Spec{
			&goast.ImportSpec{
				Path: &goast.BasicLit{
					Kind:  goasttoken.STRING,
					Value: `"encoding/json"`,
				},
			},
		},
	})
}

func (t *Transformer) NodeRuntimeFile() (*goast.File, error) {
	if t == nil || t.NodeRuntimeOutput == nil || !EmitNeedsNodeRuntime(t.TypeChecker) {
		return nil, nil
	}
	return t.NodeRuntimeOutput.GenerateFile()
}

func nodeWrapperFuncName(kind, moduleID, export string) string {
	base := nodeWrapperSanitizer.ReplaceAllString(moduleID+"_"+export, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "export"
	}
	return "forst_node_" + kind + "_" + base
}

func (t *Transformer) registerNodeCallWrapper(
	target typechecker.NodeCallTarget,
	ret ast.TypeNode,
	bridgeFn string,
	e ast.FunctionCallNode,
) (string, []goast.Expr, error) {
	t.appendNodeBridgeIfNeeded()
	name := nodeWrapperFuncName(strings.ToLower(bridgeFn), target.ModuleID, target.ExportName)
	if t.nodeWrappersEmitted == nil {
		t.nodeWrappersEmitted = make(map[string]bool)
	}
	if t.nodeWrappersEmitted[name] {
		return name, t.nodeWrapperCallArgs(e), nil
	}

	valueType, err := nodeBridgeCallValueType(ret)
	if err != nil {
		return "", nil, err
	}
	retGoType, err := t.nodeBridgeResultGoType(valueType)
	if err != nil {
		return "", nil, err
	}

	bridgeFnName := bridgeFn
	var inner *goast.CallExpr
	callArgs := t.nodeWrapperCallArgs(e)
	if argsJSON, ok := t.tryEmitStaticNodeCallArgsJSONInRuntime(e.Arguments); ok {
		bridgeFnName = bridgeFn + "Args"
		inner = &goast.CallExpr{
			Fun: &goast.IndexExpr{
				X: &goast.SelectorExpr{
					X:   goast.NewIdent("nodert"),
					Sel: goast.NewIdent(bridgeFnName),
				},
				Index: retGoType,
			},
			Args: []goast.Expr{
				nodeBridgeStringLit(target.ModuleID),
				nodeBridgeStringLit(target.ExportName),
				argsJSON,
			},
		}
		callArgs = nil
	} else {
		params, paramIdents, err := t.nodeWrapperParamFieldsFromIndex(target, len(e.Arguments))
		if err != nil {
			return "", nil, err
		}
		inner = &goast.CallExpr{
			Fun: &goast.IndexExpr{
				X: &goast.SelectorExpr{
					X:   goast.NewIdent("nodert"),
					Sel: goast.NewIdent(bridgeFnName),
				},
				Index: retGoType,
			},
			Args: t.nodeBridgeCallArgsFromParamIdents(target, paramIdents),
		}
		t.nodeRuntimeOut().AddFunction(&goast.FuncDecl{
			Name: goast.NewIdent(name),
			Type: &goast.FuncType{
				Params: &goast.FieldList{List: params},
				Results: &goast.FieldList{List: []*goast.Field{
					{Type: retGoType},
					{Type: goast.NewIdent("error")},
				}},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{inner}}},
			},
		})
		t.nodeWrappersEmitted[name] = true
		return name, callArgs, nil
	}

	t.nodeRuntimeOut().AddFunction(&goast.FuncDecl{
		Name: goast.NewIdent(name),
		Type: &goast.FuncType{
			Results: &goast.FieldList{List: []*goast.Field{
				{Type: retGoType},
				{Type: goast.NewIdent("error")},
			}},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{&goast.ReturnStmt{Results: []goast.Expr{inner}}},
		},
	})
	t.nodeWrappersEmitted[name] = true
	return name, callArgs, nil
}

func (t *Transformer) nodeWrapperCallArgs(e ast.FunctionCallNode) []goast.Expr {
	args := make([]goast.Expr, 0, len(e.Arguments))
	for _, arg := range e.Arguments {
		expr, err := t.transformExpression(arg)
		if err != nil {
			panic(err)
		}
		args = append(args, expr)
	}
	return args
}

func (t *Transformer) nodeBridgeCallArgsFromParamIdents(target typechecker.NodeCallTarget, paramIdents []goast.Expr) []goast.Expr {
	goArgs := make([]goast.Expr, 0, len(paramIdents)+2)
	goArgs = append(goArgs,
		nodeBridgeStringLit(target.ModuleID),
		nodeBridgeStringLit(target.ExportName),
	)
	goArgs = append(goArgs, paramIdents...)
	return goArgs
}

func (t *Transformer) nodeWrapperParamFieldsFromIndex(target typechecker.NodeCallTarget, argCount int) ([]*goast.Field, []goast.Expr, error) {
	if t == nil || t.TypeChecker == nil {
		return nil, nil, fmt.Errorf("codegen: node wrapper: missing typechecker")
	}
	forstParams, err := t.TypeChecker.NodeExportParamTypes(target.ModuleID, target.ExportName)
	if err != nil {
		return nil, nil, fmt.Errorf("codegen: node wrapper %s.%s: %w", target.ModuleID, target.ExportName, err)
	}
	if len(forstParams) != argCount {
		return nil, nil, fmt.Errorf("codegen: node wrapper %s.%s: index has %d params, call has %d args",
			target.ModuleID, target.ExportName, len(forstParams), argCount)
	}
	fields := make([]*goast.Field, 0, len(forstParams))
	idents := make([]goast.Expr, 0, len(forstParams))
	for i, paramType := range forstParams {
		goType, err := t.transformType(paramType)
		if err != nil {
			return nil, nil, err
		}
		pname := goast.NewIdent(fmt.Sprintf("arg%d", i))
		fields = append(fields, &goast.Field{Names: []*goast.Ident{pname}, Type: goType})
		idents = append(idents, pname)
	}
	return fields, idents, nil
}

func (t *Transformer) tryEmitStaticNodeCallArgsJSONInRuntime(args []ast.ExpressionNode) (goast.Expr, bool) {
	expr, ok := t.tryEmitStaticNodeCallArgsJSON(args)
	if ok {
		t.ensureJSONImportInRuntime()
	}
	return expr, ok
}

func (t *Transformer) registerNodeOpenSeqWrapper(
	target typechecker.NodeCallTarget,
	ret ast.TypeNode,
	e ast.FunctionCallNode,
	async bool,
) (string, []goast.Expr, error) {
	t.appendNodeBridgeIfNeeded()
	name := nodeWrapperFuncName("open_seq", target.ModuleID, target.ExportName)
	seqTypeName, seqTypeIdent, _, err := t.ensureForstNodeSeqTypes(ret)
	if err != nil {
		return "", nil, err
	}
	_ = seqTypeName

	if t.nodeWrappersEmitted != nil && t.nodeWrappersEmitted[name] {
		return name, t.nodeWrapperCallArgs(e), nil
	}
	if t.nodeWrappersEmitted == nil {
		t.nodeWrappersEmitted = make(map[string]bool)
	}

	seqType, err := nodeBridgeCallValueType(ret)
	if err != nil {
		return "", nil, err
	}
	elemType, err := nodeSeqElementType(seqType)
	if err != nil {
		return "", nil, err
	}
	elemGoType, err := t.transformType(elemType)
	if err != nil {
		return "", nil, err
	}
	kindSel := "ExportKindGenerator"
	if async {
		kindSel = "ExportKindAsyncGenerator"
	}

	callArgs := t.nodeWrapperCallArgs(e)
	var openCall *goast.CallExpr
	if argsJSON, ok := t.tryEmitStaticNodeCallArgsJSONInRuntime(e.Arguments); ok {
		openCall = &goast.CallExpr{
			Fun: &goast.IndexExpr{
				X: &goast.SelectorExpr{X: goast.NewIdent("nodert"), Sel: goast.NewIdent("OpenSeqArgs")},
				Index: elemGoType,
			},
			Args: []goast.Expr{
				nodeBridgeStringLit(target.ModuleID),
				nodeBridgeStringLit(target.ExportName),
				&goast.SelectorExpr{X: goast.NewIdent("nodert"), Sel: goast.NewIdent(kindSel)},
				argsJSON,
			},
		}
		callArgs = nil
	} else {
		params, paramIdents, err := t.nodeWrapperParamFieldsFromIndex(target, len(e.Arguments))
		if err != nil {
			return "", nil, err
		}
		goArgs := t.nodeBridgeCallArgsFromParamIdents(target, paramIdents)
		goArgs = append(goArgs[:2], append([]goast.Expr{
			&goast.SelectorExpr{X: goast.NewIdent("nodert"), Sel: goast.NewIdent(kindSel)},
		}, goArgs[2:]...)...)
		openCall = &goast.CallExpr{
			Fun: &goast.IndexExpr{
				X: &goast.SelectorExpr{X: goast.NewIdent("nodert"), Sel: goast.NewIdent("OpenSeq")},
				Index: elemGoType,
			},
			Args: goArgs,
		}
		body := t.openSeqWrapperBody(seqTypeIdent, openCall)
		t.nodeRuntimeOut().AddFunction(&goast.FuncDecl{
			Name: goast.NewIdent(name),
			Type: &goast.FuncType{
				Params: &goast.FieldList{List: params},
				Results: &goast.FieldList{List: []*goast.Field{
					{Type: &goast.StarExpr{X: seqTypeIdent}},
					{Type: goast.NewIdent("error")},
				}},
			},
			Body: body,
		})
		t.nodeWrappersEmitted[name] = true
		return name, callArgs, nil
	}

	body := t.openSeqWrapperBody(seqTypeIdent, openCall)
	t.nodeRuntimeOut().AddFunction(&goast.FuncDecl{
		Name: goast.NewIdent(name),
		Type: &goast.FuncType{
			Results: &goast.FieldList{List: []*goast.Field{
				{Type: &goast.StarExpr{X: seqTypeIdent}},
				{Type: goast.NewIdent("error")},
			}},
		},
		Body: body,
	})
	t.nodeWrappersEmitted[name] = true
	return name, callArgs, nil
}

func (t *Transformer) openSeqWrapperBody(seqTypeIdent *goast.Ident, openCall *goast.CallExpr) *goast.BlockStmt {
	return &goast.BlockStmt{List: []goast.Stmt{
		&goast.DeclStmt{Decl: &goast.GenDecl{
			Tok: goasttoken.VAR,
			Specs: []goast.Spec{&goast.ValueSpec{
				Names:  []*goast.Ident{goast.NewIdent("seq"), goast.NewIdent("err")},
				Values: []goast.Expr{openCall},
			}},
		}},
		&goast.IfStmt{
			Cond: &goast.BinaryExpr{X: goast.NewIdent("err"), Op: goasttoken.NEQ, Y: goast.NewIdent("nil")},
			Body: &goast.BlockStmt{List: []goast.Stmt{
				&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("nil"), goast.NewIdent("err")}},
			}},
		},
		&goast.ReturnStmt{Results: []goast.Expr{
			&goast.UnaryExpr{Op: goasttoken.AND, X: &goast.CompositeLit{
				Type: seqTypeIdent,
				Elts: []goast.Expr{&goast.KeyValueExpr{Key: goast.NewIdent("inner"), Value: goast.NewIdent("seq")}},
			}},
			goast.NewIdent("nil"),
		}},
	}}
}

func (t *Transformer) ensureForstNodeSeqTypes(ret ast.TypeNode) (string, *goast.Ident, *goast.Ident, error) {
	seqType, err := nodeBridgeCallValueType(ret)
	if err != nil {
		return "", nil, nil, err
	}
	elemType, err := nodeSeqElementType(seqType)
	if err != nil {
		return "", nil, nil, err
	}
	if err := t.ensureNodeReturnTypeEmitted(elemType); err != nil {
		return "", nil, nil, err
	}
	elemGoType, err := t.transformType(elemType)
	if err != nil {
		return "", nil, nil, err
	}
	typeSuffix := forstNodeSeqTypeSuffix(elemGoType)
	seqName := forstNodeSeqTypePrefix + typeSuffix
	stepName := forstNodeGenStepTypePrefix + typeSuffix
	seqIdent := goast.NewIdent(seqName)
	stepIdent := goast.NewIdent(stepName)
	if t.nodeSeqTypesEmitted == nil {
		t.nodeSeqTypesEmitted = make(map[string]bool)
	}
	if !t.nodeSeqTypesEmitted[seqName] {
		out := t.nodeRuntimeOut()
		out.AddType(&goast.GenDecl{
			Tok: goasttoken.TYPE,
			Specs: []goast.Spec{
				&goast.TypeSpec{
					Name: stepIdent,
					Type: &goast.StructType{Fields: &goast.FieldList{List: []*goast.Field{
						{Names: []*goast.Ident{goast.NewIdent("Kind")}, Type: goast.NewIdent("string")},
						{Names: []*goast.Ident{goast.NewIdent("Value")}, Type: elemGoType},
						{Names: []*goast.Ident{goast.NewIdent("Message")}, Type: goast.NewIdent("string")},
					}}},
				},
				&goast.TypeSpec{
					Name: seqIdent,
					Type: &goast.StructType{Fields: &goast.FieldList{List: []*goast.Field{
						{Names: []*goast.Ident{goast.NewIdent("inner")}, Type: &goast.StarExpr{X: &goast.IndexExpr{
							X:     &goast.SelectorExpr{X: goast.NewIdent("nodert"), Sel: goast.NewIdent("Seq")},
							Index: elemGoType,
						}}},
					}}},
				},
			},
		})
		t.emitForstNodeSeqMethods(seqIdent, stepIdent, elemGoType)
		t.nodeSeqTypesEmitted[seqName] = true
	}
	return seqName, seqIdent, stepIdent, nil
}

func forstNodeSeqTypeSuffix(elemGoType goast.Expr) string {
	switch e := elemGoType.(type) {
	case *goast.Ident:
		return "_" + strings.ToLower(e.Name)
	case *goast.SelectorExpr:
		if x, ok := e.X.(*goast.Ident); ok {
			return "_" + strings.ToLower(x.Name) + "_" + strings.ToLower(e.Sel.Name)
		}
	}
	return "_anon"
}

func (t *Transformer) emitForstNodeSeqMethods(seqIdent, stepIdent *goast.Ident, elemGoType goast.Expr) {
	out := t.nodeRuntimeOut()
	recv := &goast.FieldList{List: []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("s")}, Type: &goast.StarExpr{X: seqIdent}}}}
	stepSlice := &goast.ArrayType{Elt: stepIdent}
	nodertGenStep := &goast.IndexExpr{
		X:     &goast.SelectorExpr{X: goast.NewIdent("nodert"), Sel: goast.NewIdent("GenStep")},
		Index: elemGoType,
	}
	out.AddFunction(&goast.FuncDecl{
		Recv: recv,
		Name: goast.NewIdent("Close"),
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ExprStmt{X: &goast.CallExpr{
				Fun: &goast.SelectorExpr{
					X:   &goast.SelectorExpr{X: goast.NewIdent("s"), Sel: goast.NewIdent("inner")},
					Sel: goast.NewIdent("Close"),
				},
			}},
		}},
	})
	out.AddFunction(&goast.FuncDecl{
		Recv: recv,
		Name: goast.NewIdent("NextBatch"),
		Type: &goast.FuncType{
			Params: &goast.FieldList{List: []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("maxItems")}, Type: goast.NewIdent("int")}}},
			Results: &goast.FieldList{List: []*goast.Field{
				{Type: stepSlice},
				{Type: goast.NewIdent("error")},
			}},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.DeclStmt{Decl: &goast.GenDecl{
				Tok: goasttoken.VAR,
				Specs: []goast.Spec{&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("raw"), goast.NewIdent("err")},
					Values: []goast.Expr{&goast.CallExpr{
						Fun: &goast.SelectorExpr{
							X:   &goast.SelectorExpr{X: goast.NewIdent("s"), Sel: goast.NewIdent("inner")},
							Sel: goast.NewIdent("NextBatch"),
						},
						Args: []goast.Expr{goast.NewIdent("maxItems")},
					}},
				}},
			}},
			&goast.IfStmt{
				Cond: &goast.BinaryExpr{X: goast.NewIdent("err"), Op: goasttoken.NEQ, Y: goast.NewIdent("nil")},
				Body: &goast.BlockStmt{List: []goast.Stmt{
					&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("nil"), goast.NewIdent("err")}},
				}},
			},
			&goast.DeclStmt{Decl: &goast.GenDecl{
				Tok: goasttoken.VAR,
				Specs: []goast.Spec{&goast.ValueSpec{
					Names: []*goast.Ident{goast.NewIdent("out")},
					Type:  stepSlice,
				}},
			}},
			&goast.RangeStmt{
				Key:   goast.NewIdent("_"),
				Value: goast.NewIdent("step"),
				Tok:   goasttoken.DEFINE,
				X:     goast.NewIdent("raw"),
				Body: &goast.BlockStmt{List: []goast.Stmt{
					&goast.AssignStmt{
						Lhs: []goast.Expr{goast.NewIdent("out")},
						Tok: goasttoken.ASSIGN,
						Rhs: []goast.Expr{&goast.CallExpr{
							Fun: goast.NewIdent("append"),
							Args: []goast.Expr{
								goast.NewIdent("out"),
								&goast.CompositeLit{
									Type: stepIdent,
									Elts: []goast.Expr{
										&goast.KeyValueExpr{Key: goast.NewIdent("Kind"), Value: &goast.CallExpr{
											Fun: goast.NewIdent("string"),
											Args: []goast.Expr{&goast.SelectorExpr{X: goast.NewIdent("step"), Sel: goast.NewIdent("Kind")}},
										}},
										&goast.KeyValueExpr{Key: goast.NewIdent("Value"), Value: &goast.SelectorExpr{X: goast.NewIdent("step"), Sel: goast.NewIdent("Value")}},
										&goast.KeyValueExpr{Key: goast.NewIdent("Message"), Value: &goast.SelectorExpr{X: goast.NewIdent("step"), Sel: goast.NewIdent("Message")}},
									},
								},
							},
						}},
					},
				}},
			},
			&goast.ReturnStmt{Results: []goast.Expr{goast.NewIdent("out"), goast.NewIdent("nil")}},
		}},
	})
	_ = nodertGenStep
}
