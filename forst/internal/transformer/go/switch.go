package transformergo

import (
	"fmt"
	"forst/internal/ast"
	goast "go/ast"
	"go/token"
)

func (t *Transformer) transformSwitchNode(n *ast.SwitchNode) (goast.Stmt, error) {
	if n == nil {
		return nil, fmt.Errorf("nil switch node")
	}
	var init goast.Stmt
	if n.Init != nil {
		var err error
		init, err = t.transformInitPostStmt(n.Init)
		if err != nil {
			return nil, err
		}
	}
	var tag goast.Expr
	if n.Tag != nil {
		var err error
		tag, err = t.transformExpression(n.Tag)
		if err != nil {
			return nil, err
		}
	}
	body := make([]goast.Stmt, 0, len(n.Clauses))
	for _, clause := range n.Clauses {
		cc := &goast.CaseClause{}
		if !clause.IsDefault {
			for _, val := range clause.Values {
				ex, err := t.transformExpression(val)
				if err != nil {
					return nil, err
				}
				cc.List = append(cc.List, ex)
			}
		}
		for _, st := range clause.Body {
			if _, ok := st.(ast.FallthroughNode); ok {
				cc.Body = append(cc.Body, &goast.BranchStmt{Tok: token.FALLTHROUGH})
				continue
			}
			gst, err := t.transformStatement(st)
			if err != nil {
				return nil, err
			}
			cc.Body = append(cc.Body, gst)
		}
		body = append(body, cc)
	}
	return &goast.SwitchStmt{Init: init, Tag: tag, Body: &goast.BlockStmt{List: body}}, nil
}
