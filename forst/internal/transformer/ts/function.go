package transformerts

import (
	"fmt"
	"forst/internal/ast"
	"forst/internal/discovery"
	"forst/internal/typechecker"
	"strings"
)

// FunctionSignature represents a TypeScript function signature
type FunctionSignature struct {
	Name       string
	Parameters []Parameter
	ReturnType string
	// Streamable mirrors discovery (keyword/heuristic or chan T).
	Streamable bool
	// StreamingRowType is the TS type for each NDJSON data row when GenerateStreamingClients is on and return is chan T with a typable element; empty otherwise.
	StreamingRowType string
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

	// Return type: same mapping path as parameters (typechecker first, then explicit AST).
	returnType := "unknown"
	funcName := string(fn.Ident.ID)

	if sig, exists := t.TypeChecker.Functions[fn.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
		returnTypeNode := &sig.ReturnTypes[0]
		tsType, err := t.typeMapping.GetTypeScriptType(returnTypeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to get TypeScript type for function return type %s: %w", returnTypeNode.Ident, err)
		}
		returnType = tsType
	} else if len(fn.ReturnTypes) > 0 {
		returnTypeNode := &fn.ReturnTypes[0]
		tsType, err := t.typeMapping.GetTypeScriptType(returnTypeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to get TypeScript type for function return type %s: %w", returnTypeNode.Ident, err)
		}
		returnType = tsType
	}

	streamable := discovery.StreamingSupported(&fn, t.TypeChecker)
	streamingRowTS := ""
	if t.GenerateStreamingClients && streamable {
		var rt *ast.TypeNode
		if sig, exists := t.TypeChecker.Functions[fn.Ident.ID]; exists && len(sig.ReturnTypes) > 0 {
			rt = &sig.ReturnTypes[0]
		} else if len(fn.ReturnTypes) > 0 {
			rt = &fn.ReturnTypes[0]
		}
		if rt != nil {
			if elem, ok := typechecker.ChannelElementType(*rt); ok {
				ts, err := t.typeMapping.GetTypeScriptType(&elem)
				if err == nil && ts != "" && ts != "unknown" {
					streamingRowTS = ts
				}
			}
		}
	}

	// Create function signature
	signature := &FunctionSignature{
		Name:             funcName,
		Parameters:       parameters,
		ReturnType:       returnType,
		Streamable:       streamable,
		StreamingRowType: streamingRowTS,
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
	// .d.ts files require declare or export on top-level declarations; interfaces use export, match that here.
	return fmt.Sprintf("export function %s(%s): Promise<%s>;", fs.Name, paramStr, fs.ReturnType)
}

// StreamTypesDeclaration returns an extra export for the streaming API, or empty.
// Uses AsyncGenerator + StreamingResult from @forst/sidecar so row typing matches the runtime NDJSON envelope.
func (fs *FunctionSignature) StreamTypesDeclaration() string {
	if fs.StreamingRowType == "" {
		return ""
	}
	params := make([]string, len(fs.Parameters))
	for i, param := range fs.Parameters {
		params[i] = fmt.Sprintf("%s: %s", param.Name, param.Type)
	}
	paramStr := strings.Join(params, ", ")
	return fmt.Sprintf(
		"export function %sStream(%s): AsyncGenerator<import('@forst/sidecar').StreamingResult & { data?: %s }, void, undefined>;",
		fs.Name, paramStr, fs.StreamingRowType,
	)
}
