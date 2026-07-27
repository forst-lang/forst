package typechecker

import (
	"fmt"
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// labelObj records a label declared in a function body.
type labelObj struct {
	name  ast.Identifier
	used  bool
	block *labelBlock
	index int // statement index within block.stmts
	isFor bool
}

// labelBlock is one lexical block for goto/label resolution (mirrors go/types/labels.go).
type labelBlock struct {
	parent *labelBlock
	stmts  []ast.Node
}

type gotoPending struct {
	label ast.Identifier
	block *labelBlock
	index int
}

type labelChecker struct {
	tc       *TypeChecker
	all      map[ast.Identifier]*labelObj
	gotos    []gotoPending
	breaks   []ast.Identifier // labeled break/continue targets
}

func (tc *TypeChecker) pushLoopLabelScope() (restore func()) {
	saved := tc.loopLabelStack
	tc.loopLabelStack = nil
	return func() { tc.loopLabelStack = saved }
}

// checkFunctionLabels validates Go-spec label/goto rules for one function or func-lit body.
func (tc *TypeChecker) checkFunctionLabels(body []ast.Node) error {
	lc := &labelChecker{
		tc:  tc,
		all: make(map[ast.Identifier]*labelObj),
	}
	root := &labelBlock{stmts: body}
	if err := lc.walkBlock(root); err != nil {
		return err
	}
	return lc.resolve()
}

func (lc *labelChecker) walkBlock(b *labelBlock) error {
	for i, stmt := range b.stmts {
		if err := lc.walkStmt(b, i, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (lc *labelChecker) walkStmt(b *labelBlock, index int, stmt ast.Node) error {
	switch n := stmt.(type) {
	case *ast.LabeledStmtNode:
		if err := lc.declareLabel(b, index, n.Label, isForStmt(n.Stmt)); err != nil {
			return err
		}
		return lc.walkStmt(b, index, n.Stmt)
	case *ast.ForNode:
		if n.Label != nil {
			if err := lc.declareLabel(b, index, n.Label, true); err != nil {
				return err
			}
		}
		child := &labelBlock{parent: b, stmts: n.Body}
		return lc.walkBlock(child)
	case *ast.GotoNode:
		if n.Label == nil {
			return fmt.Errorf("goto requires a label")
		}
		lc.gotos = append(lc.gotos, gotoPending{label: n.Label.ID, block: b, index: index})
		return nil
	case *ast.BreakNode:
		if n.Label != nil {
			lc.breaks = append(lc.breaks, n.Label.ID)
		}
		return nil
	case *ast.ContinueNode:
		if n.Label != nil {
			lc.breaks = append(lc.breaks, n.Label.ID)
		}
		return nil
	case *ast.IfNode:
		if err := lc.walkBlock(&labelBlock{parent: b, stmts: n.Body}); err != nil {
			return err
		}
		for i := range n.ElseIfs {
			if err := lc.walkBlock(&labelBlock{parent: b, stmts: n.ElseIfs[i].Body}); err != nil {
				return err
			}
		}
		if n.Else != nil {
			if err := lc.walkStmt(b, index, n.Else); err != nil {
				return err
			}
		}
		return nil
	case *ast.ElseBlockNode:
		return lc.walkBlock(&labelBlock{parent: b, stmts: n.Body})
	case ast.ElseBlockNode:
		return lc.walkBlock(&labelBlock{parent: b, stmts: n.Body})
	case *ast.SwitchNode:
		for i := range n.Clauses {
			if err := lc.walkBlock(&labelBlock{parent: b, stmts: n.Clauses[i].Body}); err != nil {
				return err
			}
		}
		return nil
	case ast.SwitchNode:
		return lc.walkStmt(b, index, &n)
	case *ast.EnsureNode:
		if n.Block != nil {
			return lc.walkBlock(&labelBlock{parent: b, stmts: n.Block.Body})
		}
		return nil
	case ast.EnsureNode:
		return lc.walkStmt(b, index, &n)
	case *ast.WithNode:
		return lc.walkBlock(&labelBlock{parent: b, stmts: n.Body})
	case ast.WithNode:
		return lc.walkBlock(&labelBlock{parent: b, stmts: n.Body})
	default:
		// Nested function literals have their own label scope; skip their bodies here.
		lc.skipFuncLitsInNode(stmt)
		return nil
	}
}

func isForStmt(stmt ast.Node) bool {
	_, ok := stmt.(*ast.ForNode)
	if ok {
		return true
	}
	_, ok = stmt.(ast.ForNode)
	return ok
}

func (lc *labelChecker) declareLabel(b *labelBlock, index int, label *ast.Ident, isFor bool) error {
	if label == nil {
		return nil
	}
	if prev, ok := lc.all[label.ID]; ok {
		return fmt.Errorf("label %q already defined", prev.name)
	}
	lc.all[label.ID] = &labelObj{name: label.ID, block: b, index: index, isFor: isFor}
	if lc.tc.log != nil && lc.tc.log.IsLevelEnabled(logrus.DebugLevel) {
		lc.tc.log.WithFields(logrus.Fields{
			"stmt":     "label",
			"label":    label.ID,
			"resolved": true,
		}).Debug("Declared statement label")
	}
	return nil
}

func (lc *labelChecker) skipFuncLitsInNode(stmt ast.Node) {
	// Best-effort: expression statements may contain func lits; their labels are checked
	// when inferFunctionLiteral runs checkFunctionLabels on the lit body.
}

func (lc *labelChecker) resolve() error {
	for _, g := range lc.gotos {
		obj, ok := lc.all[g.label]
		if !ok {
			return fmt.Errorf("label %q not declared", g.label)
		}
		obj.used = true
		if lc.tc.log != nil {
			lc.tc.log.WithFields(logrus.Fields{
				"stmt":     "goto",
				"label":    g.label,
				"resolved": true,
			}).Debug("Resolved goto target")
		}
		if jumpsIntoBlock(g.block, obj.block) {
			return fmt.Errorf("goto %s jumps into block", g.label)
		}
		if err := checkGotoOverDecl(g, obj); err != nil {
			return err
		}
	}
	for _, name := range lc.breaks {
		obj, ok := lc.all[name]
		if !ok {
			return fmt.Errorf("undefined label %q for break/continue", name)
		}
		obj.used = true
		if !obj.isFor {
			return fmt.Errorf("invalid break/continue label %q (not a for loop)", name)
		}
	}
	for _, obj := range lc.all {
		if !obj.used {
			return fmt.Errorf("label %s defined and not used", obj.name)
		}
	}
	return nil
}

// jumpsIntoBlock reports whether a goto in from jumps into the block containing to
// (label is inside a block that does not contain the goto).
func jumpsIntoBlock(from, to *labelBlock) bool {
	if from == to {
		return false
	}
	// Label nested inside goto's block → jump into
	for b := to.parent; b != nil; b = b.parent {
		if b == from {
			return true
		}
	}
	// Goto nested inside label's block → jump out (OK)
	for b := from.parent; b != nil; b = b.parent {
		if b == to {
			return false
		}
	}
	// Sibling / unrelated blocks → jump into the label's block
	return true
}

// checkGotoOverDecl rejects forward jumps over variable declarations in shared blocks
// whose scope would include the label (Go spec).
func checkGotoOverDecl(g gotoPending, obj *labelObj) error {
	// Find the deepest common ancestor block of goto and label.
	fromDepth := blockDepth(g.block)
	toDepth := blockDepth(obj.block)
	from, to := g.block, obj.block
	for fromDepth > toDepth {
		from = from.parent
		fromDepth--
	}
	for toDepth > fromDepth {
		to = to.parent
		toDepth--
	}
	for from != to {
		from = from.parent
		to = to.parent
	}
	common := from
	if common == nil {
		return nil
	}

	// Statement index of goto/label projected into common block.
	gIdx := indexInAncestor(g.block, g.index, common)
	lIdx := indexInAncestor(obj.block, obj.index, common)
	if gIdx < 0 || lIdx < 0 || gIdx >= lIdx {
		return nil // backward jump or unresolved — no skip-over in common
	}
	// Any var decl in common between goto and label is jumped over and still in scope at label.
	for i := gIdx + 1; i <= lIdx; i++ {
		if i >= len(common.stmts) {
			break
		}
		if names := declNamesIntroduced(common.stmts[i]); len(names) > 0 {
			return fmt.Errorf("goto %s jumps over declaration of %s", g.label, names[0])
		}
	}
	return nil
}

func blockDepth(b *labelBlock) int {
	d := 0
	for ; b != nil; b = b.parent {
		d++
	}
	return d
}

// indexInAncestor returns the statement index in ancestor that contains the nested stmt.
func indexInAncestor(block *labelBlock, index int, ancestor *labelBlock) int {
	if block == ancestor {
		return index
	}
	child := block
	for child != nil && child.parent != ancestor {
		child = child.parent
	}
	if child == nil || child.parent != ancestor {
		return -1
	}
	// Find which stmt in ancestor embeds this child block — approximate by scanning.
	// Labels/gotos nest inside if/for bodies; the child block's stmts slice is that body.
	for i, stmt := range ancestor.stmts {
		if statementContainsBlock(stmt, child) {
			return i
		}
	}
	return -1
}

func statementContainsBlock(stmt ast.Node, b *labelBlock) bool {
	switch n := stmt.(type) {
	case *ast.ForNode:
		return sameStmtSlice(n.Body, b.stmts)
	case *ast.IfNode:
		if sameStmtSlice(n.Body, b.stmts) {
			return true
		}
		for i := range n.ElseIfs {
			if sameStmtSlice(n.ElseIfs[i].Body, b.stmts) {
				return true
			}
		}
		if n.Else != nil {
			return statementContainsBlock(n.Else, b)
		}
	case *ast.ElseBlockNode:
		return sameStmtSlice(n.Body, b.stmts)
	case ast.ElseBlockNode:
		return sameStmtSlice(n.Body, b.stmts)
	case *ast.SwitchNode:
		for i := range n.Clauses {
			if sameStmtSlice(n.Clauses[i].Body, b.stmts) {
				return true
			}
		}
	case *ast.LabeledStmtNode:
		return statementContainsBlock(n.Stmt, b)
	case *ast.EnsureNode:
		if n.Block != nil {
			return sameStmtSlice(n.Block.Body, b.stmts)
		}
	case *ast.WithNode:
		return sameStmtSlice(n.Body, b.stmts)
	case ast.WithNode:
		return sameStmtSlice(n.Body, b.stmts)
	}
	return false
}

func sameStmtSlice(a, b []ast.Node) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		// Empty bodies: treat as equal only when both are the same slice header (nil or shared).
		return cap(a) == cap(b) && (a == nil) == (b == nil)
	}
	return &a[0] == &b[0]
}

func declNamesIntroduced(stmt ast.Node) []string {
	switch n := stmt.(type) {
	case *ast.LabeledStmtNode:
		return declNamesIntroduced(n.Stmt)
	case ast.AssignmentNode:
		return assignmentDeclNames(n)
	case *ast.AssignmentNode:
		return assignmentDeclNames(*n)
	}
	return nil
}

func assignmentDeclNames(n ast.AssignmentNode) []string {
	// Short := always introduces names. Typed `x: T =` also declares.
	if !n.IsShort && (len(n.ExplicitTypes) == 0 || n.ExplicitTypes[0] == nil) {
		return nil
	}
	var names []string
	for _, lv := range n.LValues {
		if v, ok := lv.(ast.VariableNode); ok {
			names = append(names, string(v.Ident.ID))
		}
	}
	return names
}
