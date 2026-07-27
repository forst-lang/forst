package ast

import "fmt"

// GotoNode is a goto statement: goto Label.
type GotoNode struct {
	Label *Ident
}

func (g GotoNode) Kind() NodeKind { return NodeKindGoto }

func (g GotoNode) String() string {
	if g.Label != nil {
		return fmt.Sprintf("Goto(%s)", g.Label.ID)
	}
	return "Goto"
}

// LabeledStmtNode attaches a label to any statement (Go LabeledStmt).
// For labeled for-loops, Stmt is typically a *ForNode; ForNode.Label may also be set
// so existing labeled break/continue and emit paths keep working.
type LabeledStmtNode struct {
	Label *Ident
	Stmt  Node
}

func (l LabeledStmtNode) Kind() NodeKind { return NodeKindLabeledStmt }

func (l LabeledStmtNode) String() string {
	name := Identifier("")
	if l.Label != nil {
		name = l.Label.ID
	}
	if l.Stmt != nil {
		return fmt.Sprintf("Labeled(%s, %s)", name, l.Stmt.String())
	}
	return fmt.Sprintf("Labeled(%s)", name)
}
