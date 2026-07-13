package transformergo

import (
	goast "go/ast"
)

// AppendInvokeShutdownIfNeeded adds ForstInvokeWaitForShutdown() to func main when the invoke
// companion is emitted (same package). Call only after InvokeServerSource returns non-empty.
func (t *Transformer) AppendInvokeShutdownIfNeeded() {
	if t == nil || !t.EmbedInvokeServer || !t.isMainPackage() {
		return
	}
	appendMainShutdownCall(t, "ForstInvokeWaitForShutdown")
}

// AppendNodeHostShutdownIfNeeded adds ForstNodeWaitForShutdown() to func main for host-mode
// nodert binaries without an invoke server companion.
func (t *Transformer) AppendNodeHostShutdownIfNeeded() {
	if t == nil || !t.EmbedNodeHostMode || !t.isMainPackage() || !EmitNeedsNodeRuntime(t.TypeChecker) {
		return
	}
	appendMainShutdownCall(t, "ForstNodeWaitForShutdown")
}

func appendMainShutdownCall(t *Transformer, fnName string) {
	for _, fn := range t.Output.functions {
		if fn.Name == nil || fn.Name.Name != "main" {
			continue
		}
		if fn.Body == nil {
			fn.Body = &goast.BlockStmt{}
		}
		fn.Body.List = append(fn.Body.List, &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun: goast.NewIdent(fnName),
			},
		})
		return
	}
}
