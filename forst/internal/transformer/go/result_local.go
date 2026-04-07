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

// resultLocalSplit records how a Forst Result binding maps to Go identifiers for multi-value returns.
type resultLocalSplit struct {
	errGoName      string
	successGoNames []string // one identifier per success slot (Tuple → x0, x1, …)
}

// hasResultLocalSplitForSimpleVariable is true when vn is a plain name bound from a folded Result (x/xErr).
func (t *Transformer) hasResultLocalSplitForSimpleVariable(vn ast.VariableNode) bool {
	if strings.Contains(string(vn.Ident.ID), ".") {
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

// transformPrintBuiltinCallArgs builds Go call arguments for print/println and fmt.Print*,
// expanding a Result-split local (success slots + err) when present.
// Used from both expression and statement transforms.
func (t *Transformer) transformPrintBuiltinCallArgs(args []ast.ExpressionNode) ([]goast.Expr, error) {
	var expanded []goast.Expr
	for _, arg := range args {
		if vn, ok := arg.(ast.VariableNode); ok {
			name := string(vn.Ident.ID)
			if t != nil && t.resultLocalSplit != nil {
				if split, ok := t.resultLocalSplit[name]; ok && split.errGoName != "" && len(split.successGoNames) > 0 {
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
