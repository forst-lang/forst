package transformergo

import (
	goast "go/ast"
)

func (t *Transformer) appendInvokeShutdownIfNeeded() {
	if t == nil || !t.EmbedInvokeServer || !t.isMainPackage() {
		return
	}
	for _, fn := range t.Output.functions {
		if fn.Name == nil || fn.Name.Name != "main" {
			continue
		}
		if fn.Body == nil {
			fn.Body = &goast.BlockStmt{}
		}
		fn.Body.List = append(fn.Body.List, &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun: goast.NewIdent("ForstInvokeWaitForShutdown"),
			},
		})
		return
	}
}
