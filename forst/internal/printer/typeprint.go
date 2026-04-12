package printer

import (
	"fmt"
	"strconv"
	"strings"

	"forst/internal/ast"
)

func quoteString(s string) string {
	return strconv.Quote(s)
}

// printType renders a TypeNode as Forst source (builtins, named, pointers, assertions).
func printType(t ast.TypeNode) string {
	switch t.Ident {
	case ast.TypeInt, ast.TypeFloat, ast.TypeString, ast.TypeBool, ast.TypeVoid, ast.TypeError:
		return t.Ident.String()
	case ast.TypeArray:
		if len(t.TypeParams) > 0 {
			return "[]" + printType(t.TypeParams[0])
		}
		return "[]"
	case ast.TypeMap:
		if len(t.TypeParams) >= 2 {
			return fmt.Sprintf("Map[%s, %s]", printType(t.TypeParams[0]), printType(t.TypeParams[1]))
		}
		return "Map"
	case ast.TypePointer:
		if len(t.TypeParams) > 0 {
			return "*" + printType(t.TypeParams[0])
		}
		return "*"
	case ast.TypeAssertion:
		if t.Assertion != nil {
			return (&printer{cfg: DefaultConfig()}).formatAssertion(*t.Assertion)
		}
		return "Assertion(?)"
	case ast.TypeShape:
		if t.Assertion != nil {
			return (&printer{cfg: DefaultConfig()}).formatAssertion(*t.Assertion)
		}
		return "Shape"
	case ast.TypeImplicit:
		return ""
	case ast.TypeResult:
		if len(t.TypeParams) >= 2 {
			return fmt.Sprintf("Result(%s, %s)", printType(t.TypeParams[0]), printType(t.TypeParams[1]))
		}
		return "Result"
	case ast.TypeTuple:
		if len(t.TypeParams) > 0 {
			ps := make([]string, len(t.TypeParams))
			for i := range t.TypeParams {
				ps[i] = printType(t.TypeParams[i])
			}
			return "Tuple(" + strings.Join(ps, ", ") + ")"
		}
		return "Tuple"
	default:
		if t.Assertion != nil {
			return string(t.Ident) + "." + (&printer{cfg: DefaultConfig()}).formatAssertionChainOnly(*t.Assertion)
		}
		if len(t.TypeParams) > 0 {
			ps := make([]string, len(t.TypeParams))
			for i := range t.TypeParams {
				ps[i] = printType(t.TypeParams[i])
			}
			return string(t.Ident) + "<" + strings.Join(ps, ", ") + ">"
		}
		return string(t.Ident)
	}
}

func (p *printer) formatAssertion(a ast.AssertionNode) string {
	if a.BaseType == nil {
		return p.formatAssertionChainOnly(a)
	}
	base := a.BaseType.String()
	if len(a.Constraints) == 0 {
		return base
	}
	return base + "." + p.formatAssertionChainOnly(a)
}

// formatAssertionChainOnly prints constraint1.constraint2 without base type.
func (p *printer) formatAssertionChainOnly(a ast.AssertionNode) string {
	parts := make([]string, len(a.Constraints))
	for i, c := range a.Constraints {
		parts[i] = p.formatConstraint(c)
	}
	return strings.Join(parts, ".")
}

func (p *printer) formatConstraint(c ast.ConstraintNode) string {
	// Single shape argument: `Input { ... }` instead of `Input({ ... })` (Forst surface syntax).
	if len(c.Args) == 1 && c.Args[0].Shape != nil && c.Args[0].Value == nil && c.Args[0].Type == nil {
		s, err := p.printShape(*c.Args[0].Shape)
		if err == nil {
			return c.Name + " " + s
		}
	}
	var b strings.Builder
	b.WriteString(c.Name)
	b.WriteByte('(')
	for i, arg := range c.Args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.formatConstraintArg(arg))
	}
	b.WriteByte(')')
	return b.String()
}

func (p *printer) formatConstraintArg(a ast.ConstraintArgumentNode) string {
	if a.Value != nil {
		s, err := p.printExpr((*a.Value).(ast.ExpressionNode))
		if err != nil {
			return "?"
		}
		return s
	}
	if a.Shape != nil {
		s, err := p.printShape(*a.Shape)
		if err != nil {
			return "?"
		}
		return s
	}
	if a.Type != nil {
		return printType(*a.Type)
	}
	return "?"
}
