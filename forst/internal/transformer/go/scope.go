package transformergo

import (
	"fmt"
	"forst/internal/ast"
)

func (t *Transformer) pushScope(node ast.Node) {
	newScope := t.TypeChecker.FindScope(node)
	if newScope == nil {
		panic(fmt.Sprintf("no scope found for node %v", node))
	}
	t.currentScope = newScope
}

func (t *Transformer) popScope() {
	if t.currentScope != nil {
		t.currentScope = t.currentScope.Parent
	} else {
		panic("no scope to pop from, we are already in the global scope")
	}
}
