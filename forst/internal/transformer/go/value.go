package transformergo

import (
	"fmt"

	"forst/internal/ast"
)

func expectNumberLiteral(arg ast.ValueNode) (ast.ValueNode, error) {
	if arg.Kind() != ast.NodeKindIntLiteral && arg.Kind() != ast.NodeKindFloatLiteral {
		return nil, fmt.Errorf("expected value to be a number literal, got %s", arg.Kind())
	}
	return arg, nil
}

func expectIntLiteral(arg ast.ValueNode) (ast.ValueNode, error) {
	if arg.Kind() != ast.NodeKindIntLiteral {
		return nil, fmt.Errorf("expected value to be an int literal, got %s", arg.Kind())
	}
	return arg, nil
}

func expectStringLiteral(arg ast.ValueNode) (ast.ValueNode, error) {
	if arg.Kind() != ast.NodeKindStringLiteral {
		return nil, fmt.Errorf("expected value to be a string literal, got %s", arg.Kind())
	}
	return arg, nil
}
