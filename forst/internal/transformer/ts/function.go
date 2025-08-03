package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"strings"
)

// FunctionSignature represents a TypeScript function signature
type FunctionSignature struct {
	Name       string
	Parameters []Parameter
	ReturnType string
}

// Parameter represents a TypeScript function parameter
type Parameter struct {
	Name string
	Type string
}

// FunctionTransformResult contains both the signature and definition for a function
type FunctionTransformResult struct {
	Signature  *FunctionSignature
	Definition string
}

// transformFunction converts a Forst function to TypeScript declaration and definition
func (t *TypeScriptTransformer) transformFunction(fn ast.FunctionNode) (*FunctionTransformResult, error) {
	// Generate TypeScript function signature
	parameters := []Parameter{}

	for _, param := range fn.Params {
		paramType := param.GetType()
		tsType, err := t.typeMapping.GetTypeScriptType(&paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to get TypeScript type for function param %s: %w", param.GetIdent(), err)
		}
		parameters = append(parameters, Parameter{
			Name: param.GetIdent(),
			Type: tsType,
		})
	}

	// Determine return type - first try typechecker's inferred types
	returnType := "any"
	funcName := string(fn.Ident.ID)

	// Look up function signature in typechecker
	if sig, exists := t.TypeChecker.Functions[fn.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
		// Use typechecker's inferred return types
		returnTypeNode := &sig.ReturnTypes[0]
		tsType, err := t.typeMapping.GetTypeScriptType(returnTypeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to get TypeScript type for function return type %s: %w", returnTypeNode.Ident, err)
		}
		returnType = tsType
	} else if len(fn.ReturnTypes) > 0 {
		// Fallback to AST return types if typechecker doesn't have them
		returnTypeNode := &fn.ReturnTypes[0]
		tsType, err := t.typeMapping.GetTypeScriptType(returnTypeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to get TypeScript type for function return type %s: %w", returnTypeNode.Ident, err)
		}
		returnType = tsType
	}

	// Create function signature
	signature := &FunctionSignature{
		Name:       funcName,
		Parameters: parameters,
		ReturnType: returnType,
	}

	// Generate definition for .client.ts file
	definition := fmt.Sprintf("  %s: async (args: any[]) => {\n", funcName)
	definition += fmt.Sprintf("    return client.invoke(\"%s\", \"%s\", args);\n", t.Output.PackageName, funcName)
	definition += "  },"

	return &FunctionTransformResult{
		Signature:  signature,
		Definition: definition,
	}, nil
}

// ToString converts a FunctionSignature to a TypeScript function declaration string
func (fs *FunctionSignature) ToString() string {
	params := make([]string, len(fs.Parameters))
	for i, param := range fs.Parameters {
		params[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
	}

	paramStr := strings.Join(params, ", ")
	return fmt.Sprintf("function %s(%s): Promise<%s>;", fs.Name, paramStr, fs.ReturnType)
}
