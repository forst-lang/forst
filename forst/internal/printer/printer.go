// Package printer pretty-prints parsed Forst AST back to source text.
package printer

import (
	"fmt"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/lexer"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

// Config controls layout (indentation only for now).
type Config struct {
	// Indent is one indentation level inside blocks (default "\t").
	Indent string
}

// DefaultConfig returns Config with tab indentation.
func DefaultConfig() Config {
	return Config{Indent: "\t"}
}

// FormatSource lexes and parses src then pretty-prints the AST. fileID is used for diagnostics.
func FormatSource(src, fileID string, log *logrus.Logger) (string, error) {
	l := lexer.New([]byte(src), fileID, log)
	tokens := l.Lex()
	p := parser.New(tokens, fileID, log)
	nodes, err := p.ParseFile()
	if err != nil {
		return "", err
	}
	return Print(DefaultConfig(), nodes)
}

// Print renders a parsed Forst file as source text.
func Print(cfg Config, nodes []ast.Node) (string, error) {
	if cfg.Indent == "" {
		cfg.Indent = "\t"
	}
	var p printer
	p.cfg = cfg
	return p.printFile(nodes)
}

// FormatTypeDefNode pretty-prints a single type definition (e.g. LSP hover).
func FormatTypeDefNode(cfg Config, def ast.TypeDefNode) (string, error) {
	if cfg.Indent == "" {
		cfg.Indent = "\t"
	}
	var p printer
	p.cfg = cfg
	return p.printTypeDef(def)
}

// FormatTypeGuardNode pretty-prints a type guard declaration (e.g. LSP hover).
func FormatTypeGuardNode(cfg Config, g ast.TypeGuardNode) (string, error) {
	if cfg.Indent == "" {
		cfg.Indent = "\t"
	}
	var p printer
	p.cfg = cfg
	return p.printTypeGuard(g)
}

type printer struct {
	cfg   Config
	depth int
}

func (p *printer) push() { p.depth++ }
func (p *printer) pop()  { p.depth-- }

func (p *printer) prefix() string {
	return strings.Repeat(p.cfg.Indent, p.depth)
}

// topLevelSeparator inserts spacing between top-level declarations like gofmt: a blank line
// between declarations; consecutive or doc-adjacent line comments use a single newline.
func topLevelSeparator(prev, next ast.Node) string {
	_, prevIsComment := prev.(ast.CommentNode)
	_, nextIsComment := next.(ast.CommentNode)
	if prevIsComment && nextIsComment {
		return "\n"
	}
	if prevIsComment && !nextIsComment {
		return "\n"
	}
	if !prevIsComment && nextIsComment {
		return "\n\n"
	}
	return "\n\n"
}

func (p *printer) formatIndentedComment(text string) string {
	return strings.TrimRight(text, "\n")
}

// prefixEachLine prepends prefix to every line in s (including empty lines).
func prefixEachLine(prefix, s string) string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(prefix)
		b.WriteString(line)
	}
	return b.String()
}

func (p *printer) printFile(nodes []ast.Node) (string, error) {
	var b strings.Builder
	for i, node := range nodes {
		if i > 0 {
			b.WriteString(topLevelSeparator(nodes[i-1], node))
		}
		s, err := p.printTopLevel(node)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	out := b.String()
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

func (p *printer) printTopLevel(node ast.Node) (string, error) {
	switch n := node.(type) {
	case ast.PackageNode:
		return fmt.Sprintf("package %s", string(n.Ident.ID)), nil
	case ast.ImportNode:
		return p.printImport(n), nil
	case ast.ImportGroupNode:
		return p.printImportGroup(n), nil
	case ast.TypeDefNode:
		return p.printTypeDef(n)
	case ast.FunctionNode:
		return p.printFunction(n)
	case *ast.TypeGuardNode:
		return p.printTypeGuard(*n)
	case ast.TypeGuardNode:
		return p.printTypeGuard(n)
	case ast.CommentNode:
		return n.Text, nil
	default:
		return "", fmt.Errorf("printer: unsupported top-level node %T", node)
	}
}

func (p *printer) printImport(i ast.ImportNode) string {
	if i.SideEffectOnly {
		return fmt.Sprintf(`import _ "%s"`, i.Path)
	}
	if i.Alias != nil {
		return fmt.Sprintf(`import %s "%s"`, i.Alias.ID, i.Path)
	}
	return fmt.Sprintf(`import "%s"`, i.Path)
}

func (p *printer) printImportGroup(g ast.ImportGroupNode) string {
	var b strings.Builder
	b.WriteString("import (\n")
	p.push()
	for _, im := range g.Imports {
		b.WriteString(p.prefix())
		if im.SideEffectOnly {
			b.WriteString(`_ "` + im.Path + `"`)
		} else if im.Alias != nil {
			fmt.Fprintf(&b, `%s "%s"`, im.Alias.ID, im.Path)
		} else {
			fmt.Fprintf(&b, `"%s"`, im.Path)
		}
		b.WriteByte('\n')
	}
	p.pop()
	b.WriteString(")\n")
	return b.String()
}

func (p *printer) printTypeDef(t ast.TypeDefNode) (string, error) {
	if _, ok := t.Expr.(ast.TypeDefErrorExpr); ok {
		ee := t.Expr.(ast.TypeDefErrorExpr)
		body, err := p.printShape(ee.Payload)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("error %s %s", string(t.Ident), body), nil
	}
	expr, err := p.printTypeDefExpr(t.Expr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("type %s = %s", string(t.Ident), expr), nil
}

func (p *printer) printTypeDefExpr(e ast.TypeDefExpr) (string, error) {
	switch x := e.(type) {
	case *ast.TypeDefAssertionExpr:
		if x == nil || x.Assertion == nil {
			return "", fmt.Errorf("printer: empty type def assertion")
		}
		return p.formatAssertion(*x.Assertion), nil
	case ast.TypeDefAssertionExpr:
		if x.Assertion == nil {
			return "", fmt.Errorf("printer: empty type def assertion")
		}
		return p.formatAssertion(*x.Assertion), nil
	case ast.TypeDefBinaryExpr:
		left, err := p.printTypeDefExpr(x.Left)
		if err != nil {
			return "", err
		}
		right, err := p.printTypeDefExpr(x.Right)
		if err != nil {
			return "", err
		}
		op := "&"
		if x.Op == ast.TokenBitwiseOr {
			op = "|"
		}
		return left + " " + op + " " + right, nil
	case ast.TypeDefShapeExpr:
		return p.printShape(x.Shape)
	case ast.TypeDefErrorExpr:
		body, err := p.printShape(x.Payload)
		if err != nil {
			return "", err
		}
		return "error " + body, nil
	default:
		return "", fmt.Errorf("printer: unsupported type def expr %T", e)
	}
}

func (p *printer) printFunction(fn ast.FunctionNode) (string, error) {
	var b strings.Builder
	b.WriteString("func ")
	b.WriteString(string(fn.Ident.ID))
	b.WriteByte('(')
	for i, param := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		ps, err := p.printParam(param)
		if err != nil {
			return "", err
		}
		b.WriteString(ps)
	}
	b.WriteByte(')')
	if len(fn.ReturnTypes) > 0 {
		b.WriteString(": ")
		for i, rt := range fn.ReturnTypes {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(printType(rt))
		}
	}
	b.WriteString(" {\n")
	p.push()
	body, err := p.printBlock(fn.Body)
	if err != nil {
		return "", err
	}
	b.WriteString(body)
	p.pop()
	b.WriteString(p.prefix())
	b.WriteByte('}')
	return b.String(), nil
}

func (p *printer) printTypeGuard(tg ast.TypeGuardNode) (string, error) {
	var b strings.Builder
	b.WriteString("is (")
	subj, err := p.printParam(tg.Subject)
	if err != nil {
		return "", err
	}
	b.WriteString(subj)
	for _, param := range tg.Params {
		b.WriteString(", ")
		ps, err := p.printParam(param)
		if err != nil {
			return "", err
		}
		b.WriteString(ps)
	}
	b.WriteString(") ")
	b.WriteString(string(tg.Ident))
	b.WriteString(" {\n")
	p.push()
	body, err := p.printBlock(tg.Body)
	if err != nil {
		return "", err
	}
	b.WriteString(body)
	p.pop()
	b.WriteString(p.prefix())
	b.WriteByte('}')
	return b.String(), nil
}

func (p *printer) printParam(param ast.ParamNode) (string, error) {
	switch x := param.(type) {
	case ast.SimpleParamNode:
		return string(x.Ident.ID) + " " + printType(x.Type), nil
	case ast.DestructuredParamNode:
		var b strings.Builder
		b.WriteByte('{')
		for i, f := range x.Fields {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(f)
		}
		b.WriteString("} ")
		b.WriteString(printType(x.Type))
		return b.String(), nil
	default:
		return "", fmt.Errorf("printer: unsupported param %T", param)
	}
}

func (p *printer) printBlock(nodes []ast.Node) (string, error) {
	var b strings.Builder
	for _, node := range nodes {
		line, err := p.printStmt(node)
		if err != nil {
			return "", err
		}
		b.WriteString(prefixEachLine(p.prefix(), line))
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func (p *printer) printStmt(node ast.Node) (string, error) {
	switch n := node.(type) {
	case ast.CommentNode:
		return p.formatIndentedComment(n.Text), nil
	case ast.AssignmentNode:
		return p.printAssignment(n)
	case ast.ReturnNode:
		return p.printReturn(n)
	case ast.EnsureNode:
		return p.printEnsure(n)
	case *ast.IfNode:
		return p.printIf(n)
	case ast.IfNode:
		return p.printIf(&n)
	case *ast.ForNode:
		return p.printFor(*n)
	case ast.ForNode:
		return p.printFor(n)
	case *ast.BreakNode:
		if n.Label != nil {
			return "break " + string(n.Label.ID), nil
		}
		return "break", nil
	case *ast.ContinueNode:
		if n.Label != nil {
			return "continue " + string(n.Label.ID), nil
		}
		return "continue", nil
	case *ast.DeferNode:
		s, err := p.printExpr(n.Call)
		if err != nil {
			return "", err
		}
		return "defer " + s, nil
	case *ast.GoStmtNode:
		s, err := p.printExpr(n.Call)
		if err != nil {
			return "", err
		}
		return "go " + s, nil
	default:
		// Expression statement
		if e, ok := node.(ast.ExpressionNode); ok {
			s, err := p.printExpr(e)
			if err != nil {
				return "", err
			}
			return s, nil
		}
		return "", fmt.Errorf("printer: unsupported statement %T", node)
	}
}

func (p *printer) printAssignment(a ast.AssignmentNode) (string, error) {
	isVarDecl := !a.IsShort && len(a.RValues) == 0 && len(a.LValues) == 1 &&
		len(a.ExplicitTypes) > 0 && a.ExplicitTypes[0] != nil

	lhsParts := make([]string, len(a.LValues))
	for i, lv := range a.LValues {
		var s string
		switch v := lv.(type) {
		case ast.VariableNode:
			s = string(v.Ident.ID)
			if len(a.ExplicitTypes) > i && a.ExplicitTypes[i] != nil {
				s += ": " + printType(*a.ExplicitTypes[i])
			} else if v.ExplicitType.Ident != "" && !v.ExplicitType.IsImplicit() {
				s += ": " + printType(v.ExplicitType)
			}
		default:
			var err error
			s, err = p.printExpr(lv)
			if err != nil {
				return "", err
			}
		}
		lhsParts[i] = s
	}
	lhs := strings.Join(lhsParts, ", ")

	if a.IsShort {
		rhs, err := p.printExprList(a.RValues)
		if err != nil {
			return "", err
		}
		return lhs + " := " + rhs, nil
	}

	if len(a.RValues) == 0 {
		if isVarDecl {
			return "var " + lhs, nil
		}
		return lhs, nil
	}

	rhs, err := p.printExprList(a.RValues)
	if err != nil {
		return "", err
	}
	// var x: T = e
	if !a.IsShort && len(a.LValues) == 1 && len(a.ExplicitTypes) > 0 && a.ExplicitTypes[0] != nil {
		return "var " + lhs + " = " + rhs, nil
	}
	return lhs + " = " + rhs, nil
}

func (p *printer) printExprList(xs []ast.ExpressionNode) (string, error) {
	parts := make([]string, len(xs))
	for i, x := range xs {
		s, err := p.printExpr(x)
		if err != nil {
			return "", err
		}
		parts[i] = s
	}
	return strings.Join(parts, ", "), nil
}

func (p *printer) printReturn(r ast.ReturnNode) (string, error) {
	if len(r.Values) == 0 {
		return "return", nil
	}
	s, err := p.printExprList(r.Values)
	if err != nil {
		return "", err
	}
	return "return " + s, nil
}

func (p *printer) printEnsure(e ast.EnsureNode) (string, error) {
	var b strings.Builder
	v := string(e.Variable.Ident.ID)
	if e.Variable.Ident.ID != "" {
		// Negated ensure !x
		// Parser stores negation as special assertion; detect by single constraint Nil on error type
		if len(e.Assertion.Constraints) == 1 && e.Assertion.Constraints[0].Name == "Nil" &&
			e.Assertion.BaseType != nil && *e.Assertion.BaseType == ast.TypeError {
			b.WriteString("ensure !")
			b.WriteString(v)
		} else {
			b.WriteString("ensure ")
			b.WriteString(v)
			b.WriteString(" is ")
			b.WriteString(p.formatAssertion(e.Assertion))
		}
	}
	if e.Error != nil {
		b.WriteString(" or ")
		switch err := (*e.Error).(type) {
		case ast.EnsureErrorCall:
			b.WriteString(err.ErrorType)
			b.WriteByte('(')
			for i, a := range err.ErrorArgs {
				if i > 0 {
					b.WriteString(", ")
				}
				s, e2 := p.printExpr(a)
				if e2 != nil {
					return "", e2
				}
				b.WriteString(s)
			}
			b.WriteByte(')')
		case ast.EnsureErrorVar:
			b.WriteString(string(err))
		default:
			return "", fmt.Errorf("printer: unknown ensure error %T", err)
		}
	}
	if e.Block != nil && len(e.Block.Body) > 0 {
		b.WriteString(" {\n")
		p.push()
		body, err := p.printBlock(e.Block.Body)
		if err != nil {
			return "", err
		}
		b.WriteString(body)
		p.pop()
		b.WriteString(p.prefix())
		b.WriteByte('}')
	}
	return b.String(), nil
}

func (p *printer) printIf(n *ast.IfNode) (string, error) {
	var b strings.Builder
	b.WriteString("if ")
	if n.Init != nil {
		initStr, err := p.printStmt(n.Init)
		if err != nil {
			return "", err
		}
		b.WriteString(initStr)
		b.WriteString("; ")
	}
	cond, err := p.printExprFromNode(n.Condition)
	if err != nil {
		return "", err
	}
	b.WriteString(cond)
	b.WriteString(" {\n")
	p.push()
	thenBody, err := p.printBlock(n.Body)
	if err != nil {
		return "", err
	}
	b.WriteString(thenBody)
	p.pop()
	b.WriteString(p.prefix())
	b.WriteByte('}')
	for _, ei := range n.ElseIfs {
		b.WriteString(" else if ")
		ec, err := p.printExprFromNode(ei.Condition)
		if err != nil {
			return "", err
		}
		b.WriteString(ec)
		b.WriteString(" {\n")
		p.push()
		eb, err := p.printBlock(ei.Body)
		if err != nil {
			return "", err
		}
		b.WriteString(eb)
		p.pop()
		b.WriteString(p.prefix())
		b.WriteByte('}')
	}
	if n.Else != nil {
		b.WriteString(" else {\n")
		p.push()
		eb, err := p.printBlock(n.Else.Body)
		if err != nil {
			return "", err
		}
		b.WriteString(eb)
		p.pop()
		b.WriteString(p.prefix())
		b.WriteByte('}')
	}
	return b.String(), nil
}

func (p *printer) printFor(n ast.ForNode) (string, error) {
	var b strings.Builder
	if n.IsRange {
		b.WriteString("for ")
		if n.RangeKey != nil {
			b.WriteString(string(n.RangeKey.ID))
			if n.RangeValue != nil {
				b.WriteString(", ")
				b.WriteString(string(n.RangeValue.ID))
			}
			if n.RangeShort {
				b.WriteString(" := ")
			} else {
				b.WriteString(" = ")
			}
		}
		b.WriteString("range ")
		rx, err := p.printExpr(n.RangeX)
		if err != nil {
			return "", err
		}
		b.WriteString(rx)
		b.WriteString(" {\n")
		p.push()
		body, err := p.printBlock(n.Body)
		if err != nil {
			return "", err
		}
		b.WriteString(body)
		p.pop()
		b.WriteString(p.prefix())
		b.WriteByte('}')
		return b.String(), nil
	}

	b.WriteString("for ")
	if n.Init != nil || n.Post != nil {
		if n.Init != nil {
			is, err := p.printStmt(n.Init)
			if err != nil {
				return "", err
			}
			b.WriteString(is)
		}
		b.WriteString("; ")
		if n.Cond != nil {
			c, err := p.printExpr(n.Cond)
			if err != nil {
				return "", err
			}
			b.WriteString(c)
		}
		b.WriteString("; ")
		if n.Post != nil {
			ps, err := p.printStmt(n.Post)
			if err != nil {
				return "", err
			}
			b.WriteString(ps)
		}
	} else if n.Cond != nil {
		c, err := p.printExpr(n.Cond)
		if err != nil {
			return "", err
		}
		b.WriteString(c)
	}
	b.WriteString(" {\n")
	p.push()
	body, err := p.printBlock(n.Body)
	if err != nil {
		return "", err
	}
	b.WriteString(body)
	p.pop()
	b.WriteString(p.prefix())
	b.WriteByte('}')
	return b.String(), nil
}

func (p *printer) printExpr(e ast.ExpressionNode) (string, error) {
	switch x := e.(type) {
	case ast.VariableNode:
		return string(x.Ident.ID), nil
	case ast.IntLiteralNode:
		return fmt.Sprintf("%d", x.Value), nil
	case ast.FloatLiteralNode:
		return fmt.Sprintf("%g", x.Value), nil
	case ast.StringLiteralNode:
		return quoteString(x.Value), nil
	case ast.BoolLiteralNode:
		if x.Value {
			return "true", nil
		}
		return "false", nil
	case ast.NilLiteralNode:
		return "nil", nil
	case ast.UnaryExpressionNode:
		inner, err := p.printExpr(x.Operand)
		if err != nil {
			return "", err
		}
		return tokenUnary(x.Operator) + inner, nil
	case ast.BinaryExpressionNode:
		return p.printBinary(x)
	case ast.FunctionCallNode:
		return p.printCall(x)
	case ast.IndexExpressionNode:
		tgt, err := p.printExpr(x.Target)
		if err != nil {
			return "", err
		}
		idx, err := p.printExpr(x.Index)
		if err != nil {
			return "", err
		}
		return tgt + "[" + idx + "]", nil
	case ast.ReferenceNode:
		inner, err := p.printValue(x.Value)
		if err != nil {
			return "", err
		}
		return "&" + inner, nil
	case ast.DereferenceNode:
		inner, err := p.printValue(x.Value)
		if err != nil {
			return "", err
		}
		return "*" + inner, nil
	case ast.ShapeNode:
		return p.printShape(x)
	case ast.ArrayLiteralNode:
		var buf strings.Builder
		buf.WriteByte('[')
		for i, lit := range x.Value {
			if i > 0 {
				buf.WriteString(", ")
			}
			s, err := p.printExpr(lit.(ast.ExpressionNode))
			if err != nil {
				return "", err
			}
			buf.WriteString(s)
		}
		buf.WriteByte(']')
		return buf.String(), nil
	case ast.MapLiteralNode:
		return "", fmt.Errorf("printer: MapLiteral not supported yet")
	default:
		return "", fmt.Errorf("printer: unsupported expression %T", e)
	}
}

func (p *printer) printValue(v ast.ValueNode) (string, error) {
	return p.printExpr(v.(ast.ExpressionNode))
}

func (p *printer) printBinary(b ast.BinaryExpressionNode) (string, error) {
	left, err := p.printExpr(b.Left)
	if err != nil {
		return "", err
	}
	right, err := p.printExpr(b.Right)
	if err != nil {
		return "", err
	}
	return left + " " + tokenBinary(b.Operator) + " " + right, nil
}

func (p *printer) printCall(c ast.FunctionCallNode) (string, error) {
	var b strings.Builder
	b.WriteString(string(c.Function.ID))
	b.WriteByte('(')
	for i, a := range c.Arguments {
		if i > 0 {
			b.WriteString(", ")
		}
		s, err := p.printExpr(a)
		if err != nil {
			return "", err
		}
		b.WriteString(s)
	}
	b.WriteByte(')')
	return b.String(), nil
}

// shapeUseMultiline picks a multi-line layout for non-empty shapes so nested structure
// stays readable (matches common hand-formatted Forst).
func shapeUseMultiline(s ast.ShapeNode) bool {
	return len(s.Fields) > 0
}

func sortedShapeFieldNames(fields map[string]ast.ShapeFieldNode) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *printer) printShape(s ast.ShapeNode) (string, error) {
	return p.printShapeAtFieldIndent(s, 1)
}

// printShapeAtFieldIndent prints a shape; fieldIndent is the number of cfg.Indent units
// used for each field line inside this shape (1 = one level of indentation inside the shape).
func (p *printer) printShapeAtFieldIndent(s ast.ShapeNode, fieldIndent int) (string, error) {
	if !shapeUseMultiline(s) {
		return p.printShapeOneLine(s)
	}
	return p.printShapeMultiline(s, fieldIndent)
}

// shapeFieldParsedValueExpr reports the expression for a shape *literal* field. The parser stores
// literal RHS values as Assertion Value(expr) for the typechecker; the printer must emit the
// original expression (e.g. nil, 1, "x") instead of Value(nil).
func shapeFieldParsedValueExpr(field ast.ShapeFieldNode) (ast.ExpressionNode, bool) {
	a := field.Assertion
	if a == nil || a.BaseType != nil || len(a.Constraints) != 1 {
		return nil, false
	}
	c := a.Constraints[0]
	if c.Name != ast.ValueConstraint || len(c.Args) != 1 {
		return nil, false
	}
	arg := c.Args[0]
	if arg.Value == nil {
		return nil, false
	}
	expr, ok := (*arg.Value).(ast.ExpressionNode)
	return expr, ok
}

func (p *printer) printShapeFieldRHS(field ast.ShapeFieldNode, fieldIndent int) (string, error) {
	if field.Shape != nil {
		return p.printShapeAtFieldIndent(*field.Shape, fieldIndent)
	}
	if expr, ok := shapeFieldParsedValueExpr(field); ok {
		return p.printExpr(expr)
	}
	if field.Assertion != nil {
		return p.formatAssertion(*field.Assertion), nil
	}
	if field.Type != nil {
		return printType(*field.Type), nil
	}
	return "", nil
}

func (p *printer) printShapeOneLine(s ast.ShapeNode) (string, error) {
	var b strings.Builder
	// Anonymous structural types use BaseType TYPE_SHAPE as a sentinel; omit it in source.
	if s.BaseType != nil && *s.BaseType != ast.TypeShape {
		b.WriteString(string(*s.BaseType))
	}
	b.WriteByte('{')
	names := sortedShapeFieldNames(s.Fields)
	for i, name := range names {
		if i > 0 {
			b.WriteString(", ")
		}
		field := s.Fields[name]
		b.WriteString(name)
		b.WriteString(": ")
		rhs, err := p.printShapeFieldRHS(field, 1)
		if err != nil {
			return "", err
		}
		b.WriteString(rhs)
	}
	b.WriteByte('}')
	return b.String(), nil
}

func (p *printer) printShapeMultiline(s ast.ShapeNode, fieldIndent int) (string, error) {
	if fieldIndent < 1 {
		fieldIndent = 1
	}
	fi := strings.Repeat(p.cfg.Indent, fieldIndent)
	closeIndent := strings.Repeat(p.cfg.Indent, fieldIndent-1)

	var b strings.Builder
	// Anonymous structural types use BaseType TYPE_SHAPE as a sentinel; omit it in source.
	if s.BaseType != nil && *s.BaseType != ast.TypeShape {
		b.WriteString(string(*s.BaseType))
	}
	b.WriteString("{\n")
	names := sortedShapeFieldNames(s.Fields)
	for _, name := range names {
		field := s.Fields[name]
		b.WriteString(fi)
		b.WriteString(name)
		b.WriteString(": ")
		rhs, err := p.printShapeFieldRHS(field, fieldIndent+1)
		if err != nil {
			return "", err
		}
		b.WriteString(rhs)
		b.WriteString(",\n")
	}
	b.WriteString(closeIndent)
	b.WriteByte('}')
	return b.String(), nil
}
