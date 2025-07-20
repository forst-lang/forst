package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

// transformFunction converts a Forst function to TypeScript declaration
func (t *TypeScriptTransformer) transformFunction(fn ast.FunctionNode) (string, error) {
	// Generate TypeScript function signature
	params := []string{}

	for _, param := range fn.Params {
		paramType := param.GetType()
		tsType, err := t.typeMapping.GetTypeScriptType(&paramType)
		if err != nil {
			return "", fmt.Errorf("failed to get TypeScript type for function param %s: %w", param.GetIdent(), err)
		}
		params = append(params, fmt.Sprintf("%s: %s", param.GetIdent(), tsType))
	}

	// Determine return type - infer from Forst return types
	returnType := "any"
	if len(fn.ReturnTypes) > 0 {
		returnTypeNode := &fn.ReturnTypes[0]
		tsType, err := t.typeMapping.GetTypeScriptType(returnTypeNode)
		if err != nil {
			return "", fmt.Errorf("failed to get TypeScript type for function return type %s: %w", returnTypeNode.Ident, err)
		}
		returnType = tsType
	}

	// Generate function declaration with proper Promise return type
	funcName := string(fn.Ident.ID)
	paramStr := strings.Join(params, ", ")

	return fmt.Sprintf("export function %s(%s): Promise<%s>;",
		funcName, paramStr, returnType), nil
}
