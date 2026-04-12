package transformergo

import (
	"strings"

	"forst/internal/ast"
	goast "go/ast"
)

// isPrintLikeBuiltinCall reports calls that behave like Go print builtins for Result-split lowering.
func isPrintLikeBuiltinCall(id ast.Ident) bool {
	switch string(id.ID) {
	case "print", "println", "fmt.Print", "fmt.Println", "fmt.Printf":
		return true
	default:
		return false
	}
}

// goFunExprFromForstCallIdent turns "pkg.Func" into a Go selector expression; otherwise a bare ident.
func goFunExprFromForstCallIdent(id ast.Ident) goast.Expr {
	s := string(id.ID)
	if i := strings.LastIndex(s, "."); i > 0 && i < len(s)-1 {
		return &goast.SelectorExpr{
			X:   goast.NewIdent(s[:i]),
			Sel: goast.NewIdent(s[i+1:]),
		}
	}
	return goast.NewIdent(s)
}

// goExprForNarrowedResultSplitLocal maps a plain Forst name bound from lowered Result(S,F) to the Go
// identifier for the success or error slot when the typechecker narrowed the binding in the current
// scope (e.g. ensure failure block vs successor narrowing).
func (t *Transformer) goExprForNarrowedResultSplitLocal(vn ast.VariableNode, split resultLocalSplit) (goast.Expr, bool) {
	if split.errGoName == "" || len(split.successGoNames) < 1 || isDotQualifiedVariable(vn) {
		return nil, false
	}
	decl, ok := t.TypeChecker.VariableTypes[vn.Ident.ID]
	if !ok || len(decl) != 1 || !decl[0].IsResultType() || len(decl[0].TypeParams) < 2 {
		return nil, false
	}
	rt := decl[0]
	succ := rt.TypeParams[0]
	fail := rt.TypeParams[1]
	lt, err := t.TypeChecker.LookupVariableType(&vn, t.currentScope())
	if err != nil {
		return nil, false
	}
	if lt.IsResultType() {
		return nil, false
	}
	if t.TypeChecker.IsTypeCompatible(lt, succ) {
		return goast.NewIdent(split.successGoNames[0]), true
	}
	if t.TypeChecker.IsTypeCompatible(lt, fail) {
		return goast.NewIdent(split.errGoName), true
	}
	return nil, false
}

// goFunExprFromForstCallIdentWithNarrowing is like goFunExprFromForstCallIdent but rewrites the
// receiver of recv.method when recv is a Result-split local and the current scope narrowed it.
func (t *Transformer) goFunExprFromForstCallIdentWithNarrowing(id ast.Ident) goast.Expr {
	s := string(id.ID)
	i := strings.LastIndex(s, ".")
	if i <= 0 || i >= len(s)-1 {
		return goFunExprFromForstCallIdent(id)
	}
	recv := s[:i]
	method := s[i+1:]
	if strings.Contains(recv, ".") {
		return goFunExprFromForstCallIdent(id)
	}
	if t.resultLocalSplit != nil {
		if split, ok := t.resultLocalSplit[recv]; ok {
			vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(recv)}}
			if expr, ok2 := t.goExprForNarrowedResultSplitLocal(vn, split); ok2 {
				return &goast.SelectorExpr{X: expr, Sel: goast.NewIdent(method)}
			}
		}
	}
	return goFunExprFromForstCallIdent(id)
}

// resultLocalSplit records how a Forst Result binding maps to Go identifiers for multi-value returns.
type resultLocalSplit struct {
	errGoName      string
	successGoNames []string // one identifier per success slot (Tuple → x0, x1, …)
}

// hasResultLocalSplitForSimpleVariable is true when vn is a plain name bound from a folded Result (x/xErr).
func (t *Transformer) hasResultLocalSplitForSimpleVariable(vn ast.VariableNode) bool {
	if isDotQualifiedVariable(vn) {
		return false
	}
	name := string(vn.Ident.ID)
	if t.resultLocalSplit == nil {
		return false
	}
	s, ok := t.resultLocalSplit[name]
	return ok && s.errGoName != ""
}

// rhsCallIsFoldedResult reports whether fc is typed as a single Forst Result (including Go (T, error) FFI).
func (t *Transformer) rhsCallIsFoldedResult(fc ast.FunctionCallNode) bool {
	ts, err := t.TypeChecker.LookupInferredType(fc, false)
	if err != nil || len(ts) != 1 || !ts[0].IsResultType() {
		return false
	}
	return true
}

// returnValueDelegatesWholeResult is true when `return expr` should forward a Result-returning
// callee as a single Go return (`return g()`), not `return succ, nil` for constructor-free success.
func (t *Transformer) returnValueDelegatesWholeResult(expr ast.ExpressionNode) bool {
	fc, ok := expr.(ast.FunctionCallNode)
	if !ok {
		return false
	}
	return t.rhsCallIsFoldedResult(fc)
}

// transformPrintBuiltinCallArgs builds Go call arguments for print/println and fmt.Print*,
// expanding a Result-split local (success slots + err) when present.
// Used from both expression and statement transforms.
func (t *Transformer) transformPrintBuiltinCallArgs(args []ast.ExpressionNode) ([]goast.Expr, error) {
	expanded := make([]goast.Expr, 0, len(args)*2)
	for _, arg := range args {
		if vn, ok := arg.(ast.VariableNode); ok {
			// Struct-stored Result fields lower to {V, Err}; expand print args like x,xErr only
			// when this occurrence is still typed as Result. When narrowed (e.g. inside `if w.r is Ok()`),
			// use normal codegen (e.g. println(w.r.V)).
			if sel, okSel := t.compoundResultFieldStorageSelector(vn); okSel {
				if occ, okOcc := t.TypeChecker.InferredTypesForVariableNode(vn); okOcc && len(occ) == 1 && occ[0].IsResultType() {
					expanded = append(expanded,
						goLoweredResultValueSelector(sel),
						goLoweredResultErrSelector(sel),
					)
					continue
				}
			}
			name := string(vn.Ident.ID)
			if t != nil && t.resultLocalSplit != nil {
				if split, ok := t.resultLocalSplit[name]; ok && split.errGoName != "" && len(split.successGoNames) > 0 {
					lt, err := t.TypeChecker.LookupVariableType(&vn, t.currentScope())
					if err == nil && !lt.IsResultType() {
						// Success-only / failure-only narrowing (e.g. after `ensure x is Ok()`): one Go value.
						argExpr, err2 := t.transformExpression(arg)
						if err2 != nil {
							return nil, err2
						}
						expanded = append(expanded, argExpr)
						continue
					}
					for _, id := range split.successGoNames {
						expanded = append(expanded, goast.NewIdent(id))
					}
					expanded = append(expanded, goast.NewIdent(split.errGoName))
					continue
				}
			}
		}
		argExpr, err := t.transformExpression(arg)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, argExpr)
	}
	return expanded, nil
}
