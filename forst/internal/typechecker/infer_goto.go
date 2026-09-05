package typechecker

import (
	"forst/internal/ast"

	logrus "github.com/sirupsen/logrus"
)

// LabelBinding is a label declaration with use sites, persisted for LSP after typecheck.
type LabelBinding struct {
	Name     ast.Identifier
	DeclSpan ast.SourceSpan
	IsFor    bool
	UseSpans []ast.SourceSpan
	ScopeID  int
}

// LabelScope groups labels declared in one function or function-literal body.
type LabelScope struct {
	ID        int
	StartLine int // 1-based inclusive; 0 if unknown
	EndLine   int // 1-based inclusive; 0 if unknown
	Labels    []LabelBinding
}

// labelObj records a label declared in a function body during validation.
type labelObj struct {
	ident *ast.Ident
	used  bool
	block *labelBlock
	index int
	isFor bool
}

// labelBlock is one lexical block for goto/label resolution (mirrors go/types/labels.go).
type labelBlock struct {
	parent *labelBlock
	stmts  []ast.Node
}

type gotoPending struct {
	ident *ast.Ident
	span  ast.SourceSpan
	block *labelBlock
	index int
}

type breakPending struct {
	ident *ast.Ident
}

type labelChecker struct {
	tc     *TypeChecker
	all    map[ast.Identifier]*labelObj
	gotos  []gotoPending
	breaks []breakPending
}

func (tc *TypeChecker) pushLoopLabelScope() (restore func()) {
	saved := tc.loopLabelStack
	tc.loopLabelStack = nil
	return func() { tc.loopLabelStack = saved }
}

// checkFunctionLabels validates Go-spec label/goto rules for one function or func-lit body
// and persists label bindings for LSP when validation succeeds.
func (tc *TypeChecker) checkFunctionLabels(body []ast.Node) error {
	lc := &labelChecker{
		tc:  tc,
		all: make(map[ast.Identifier]*labelObj),
	}
	root := &labelBlock{stmts: body}
	if err := lc.walkBlock(root); err != nil {
		return err
	}
	if err := lc.resolve(); err != nil {
		return err
	}
	tc.persistLabelScope(body, lc)
	return nil
}

func (tc *TypeChecker) persistLabelScope(body []ast.Node, lc *labelChecker) {
	if tc == nil || lc == nil {
		return
	}
	tc.labelScopeSeq++
	scopeID := tc.labelScopeSeq
	start, end := labelBodyLineRange(body)
	scope := LabelScope{
		ID:        scopeID,
		StartLine: start,
		EndLine:   end,
	}
	for _, obj := range lc.all {
		if obj == nil || obj.ident == nil {
			continue
		}
		b := LabelBinding{
			Name:     obj.ident.ID,
			DeclSpan: obj.ident.Span,
			IsFor:    obj.isFor,
			ScopeID:  scopeID,
		}
		for _, g := range lc.gotos {
			if g.ident != nil && g.ident.ID == obj.ident.ID {
				b.UseSpans = append(b.UseSpans, g.ident.Span)
			}
		}
		for _, br := range lc.breaks {
			if br.ident != nil && br.ident.ID == obj.ident.ID {
				b.UseSpans = append(b.UseSpans, br.ident.Span)
			}
		}
		scope.Labels = append(scope.Labels, b)
	}
	tc.LabelScopes = append(tc.LabelScopes, scope)
}

func labelBodyLineRange(body []ast.Node) (start, end int) {
	for _, stmt := range body {
		walkLabelSpans(stmt, func(sp ast.SourceSpan) {
			if !sp.IsSet() {
				return
			}
			if start == 0 || sp.StartLine < start {
				start = sp.StartLine
			}
			el := sp.EndLine
			if el == 0 {
				el = sp.StartLine
			}
			if end == 0 || el > end {
				end = el
			}
		})
	}
	return start, end
}

func walkLabelSpans(stmt ast.Node, visit func(ast.SourceSpan)) {
	switch n := stmt.(type) {
	case *ast.LabeledStmtNode:
		if n.Label != nil {
			visit(n.Label.Span)
		}
		if n.Stmt != nil {
			walkLabelSpans(n.Stmt, visit)
		}
	case *ast.GotoNode:
		if n.Label != nil {
			visit(n.Label.Span)
		}
	case *ast.BreakNode:
		if n.Label != nil {
			visit(n.Label.Span)
		}
	case *ast.ContinueNode:
		if n.Label != nil {
			visit(n.Label.Span)
		}
	case *ast.ForNode:
		if n.Label != nil {
			visit(n.Label.Span)
		}
		for _, s := range n.Body {
			walkLabelSpans(s, visit)
		}
	case *ast.IfNode:
		for _, s := range n.Body {
			walkLabelSpans(s, visit)
		}
		for i := range n.ElseIfs {
			for _, s := range n.ElseIfs[i].Body {
				walkLabelSpans(s, visit)
			}
		}
		if n.Else != nil {
			walkLabelSpans(n.Else, visit)
		}
	case *ast.ElseBlockNode:
		for _, s := range n.Body {
			walkLabelSpans(s, visit)
		}
	case ast.ElseBlockNode:
		for _, s := range n.Body {
			walkLabelSpans(s, visit)
		}
	case *ast.SwitchNode:
		for i := range n.Clauses {
			for _, s := range n.Clauses[i].Body {
				walkLabelSpans(s, visit)
			}
		}
	case ast.AssignmentNode:
		for _, lv := range n.LValues {
			if v, ok := lv.(ast.VariableNode); ok {
				visit(v.Ident.Span)
			}
		}
	case *ast.AssignmentNode:
		walkLabelSpans(*n, visit)
	}
}

// LabelBindingAtSpan returns the label binding whose decl or use span matches sp (same file/line/col).
func (tc *TypeChecker) LabelBindingAtSpan(sp ast.SourceSpan) *LabelBinding {
	if tc == nil || !sp.IsSet() {
		return nil
	}
	for i := range tc.LabelScopes {
		scope := &tc.LabelScopes[i]
		for j := range scope.Labels {
			b := &scope.Labels[j]
			if spanMatchesIdent(b.DeclSpan, sp) {
				return b
			}
			for _, u := range b.UseSpans {
				if spanMatchesIdent(u, sp) {
					return b
				}
			}
		}
	}
	return nil
}

// LabelsInScopeAt returns labels declared in the label scope that contains the given line.
func (tc *TypeChecker) LabelsInScopeAt(line int) []LabelBinding {
	if tc == nil || line <= 0 {
		return nil
	}
	var out []LabelBinding
	for i := range tc.LabelScopes {
		scope := &tc.LabelScopes[i]
		if scope.StartLine > 0 && scope.EndLine > 0 {
			if line < scope.StartLine || line > scope.EndLine {
				continue
			}
		}
		out = append(out, scope.Labels...)
	}
	return out
}

func spanMatchesIdent(a, b ast.SourceSpan) bool {
	if !a.IsSet() || !b.IsSet() {
		return false
	}
	return a.StartLine == b.StartLine && a.StartCol == b.StartCol
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
			return reportBodyf(n.Span, "goto", "goto requires a label")
		}
		lc.gotos = append(lc.gotos, gotoPending{ident: n.Label, span: n.Span, block: b, index: index})
		return nil
	case *ast.BreakNode:
		if n.Label != nil {
			lc.breaks = append(lc.breaks, breakPending{ident: n.Label})
		}
		return nil
	case *ast.ContinueNode:
		if n.Label != nil {
			lc.breaks = append(lc.breaks, breakPending{ident: n.Label})
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
		sp := label.Span
		return reportBodyf(sp, "goto", "label %q already defined", prev.ident.ID)
	}
	lc.all[label.ID] = &labelObj{ident: label, block: b, index: index, isFor: isFor}
	if lc.tc.log != nil && lc.tc.log.IsLevelEnabled(logrus.DebugLevel) {
		lc.tc.log.WithFields(logrus.Fields{
			"stmt":     "label",
			"label":    label.ID,
			"resolved": true,
		}).Debug("Declared statement label")
	}
	return nil
}

func (lc *labelChecker) resolve() error {
	for _, g := range lc.gotos {
		if g.ident == nil {
			return reportBodyf(g.span, "goto", "goto requires a label")
		}
		obj, ok := lc.all[g.ident.ID]
		if !ok {
			return reportBodyf(g.ident.Span, "goto", "label %q not declared", g.ident.ID)
		}
		obj.used = true
		if lc.tc.log != nil {
			lc.tc.log.WithFields(logrus.Fields{
				"stmt":     "goto",
				"label":    g.ident.ID,
				"resolved": true,
			}).Debug("Resolved goto target")
		}
		if jumpsIntoBlock(g.block, obj.block) {
			return reportBodyf(g.ident.Span, "goto", "goto %s jumps into block", g.ident.ID)
		}
		if err := checkGotoOverDecl(g, obj); err != nil {
			return err
		}
	}
	for _, br := range lc.breaks {
		if br.ident == nil {
			continue
		}
		obj, ok := lc.all[br.ident.ID]
		if !ok {
			return reportBodyf(br.ident.Span, "goto", "undefined label %q for break/continue", br.ident.ID)
		}
		obj.used = true
		if !obj.isFor {
			return reportBodyf(br.ident.Span, "goto", "invalid break/continue label %q (not a for loop)", br.ident.ID)
		}
	}
	for _, obj := range lc.all {
		if !obj.used {
			sp := ast.SourceSpan{}
			if obj.ident != nil {
				sp = obj.ident.Span
			}
			name := ast.Identifier("")
			if obj.ident != nil {
				name = obj.ident.ID
			}
			return reportBodyf(sp, "goto", "label %s defined and not used", name)
		}
	}
	return nil
}

// jumpsIntoBlock reports whether a goto in from jumps into the block containing to.
func jumpsIntoBlock(from, to *labelBlock) bool {
	if from == to {
		return false
	}
	for b := to.parent; b != nil; b = b.parent {
		if b == from {
			return true
		}
	}
	for b := from.parent; b != nil; b = b.parent {
		if b == to {
			return false
		}
	}
	return true
}

func checkGotoOverDecl(g gotoPending, obj *labelObj) error {
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

	gIdx := indexInAncestor(g.block, g.index, common)
	lIdx := indexInAncestor(obj.block, obj.index, common)
	if gIdx < 0 || lIdx < 0 || gIdx >= lIdx {
		return nil
	}
	for i := gIdx + 1; i <= lIdx; i++ {
		if i >= len(common.stmts) {
			break
		}
		if names := declNamesIntroduced(common.stmts[i]); len(names) > 0 {
			sp := firstSetSpan(g.ident.Span, g.span)
			return reportBodyf(sp, "goto", "goto %s jumps over declaration of %s", g.ident.ID, names[0])
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
