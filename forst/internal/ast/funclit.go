package ast

import (
	"fmt"
	"strings"
)

// TypeFunc is a function type func(Params): Returns.
const TypeFunc TypeIdent = "TYPE_FUNC"

// FunctionLiteralNode is a function literal expression: func(params): ReturnType { ... }.
type FunctionLiteralNode struct {
	Params      []ParamNode
	ReturnTypes []TypeNode
	Body        []Node
	Span        SourceSpan
}

func (FunctionLiteralNode) isExpression() {}

func (FunctionLiteralNode) Kind() NodeKind { return NodeKindFunctionLiteral }

func (n FunctionLiteralNode) String() string {
	var b strings.Builder
	b.WriteString("func(...)")
	if len(n.ReturnTypes) > 0 {
		b.WriteString(": ")
		b.WriteString(n.ReturnTypes[0].String())
	}
	b.WriteString(" { ... }")
	return b.String()
}

// NewFunctionType returns a function type from parameter and return type nodes.
func NewFunctionType(params []ParamNode, returns []TypeNode) TypeNode {
	return TypeNode{
		Ident:       TypeFunc,
		TypeKind:    TypeKindBuiltin,
		FuncParams:  append([]ParamNode(nil), params...),
		FuncReturns: append([]TypeNode(nil), returns...),
	}
}

func (t TypeNode) IsFunctionType() bool {
	return t.Ident == TypeFunc
}

func formatFuncTypeParams(params []ParamNode) string {
	parts := make([]string, len(params))
	for i, p := range params {
		switch sp := p.(type) {
		case SimpleParamNode:
			if sp.Ident.ID != "" {
				parts[i] = fmt.Sprintf("%s %s", sp.Ident.ID, sp.Type.String())
			} else {
				parts[i] = sp.Type.String()
			}
		default:
			parts[i] = p.String()
		}
	}
	return strings.Join(parts, ", ")
}
