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
	// FailureType is the TS union of domain and transport failures for this function.
	FailureType string
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

func domainErrorsByName(list []ErrorClass) map[string]ErrorClass {
	out := make(map[string]ErrorClass, len(list))
	for _, c := range list {
		out[c.Name] = c
	}
	return out
}

// clientInvokeReturnType maps a Forst return type to the TS value returned by generated invoke clients.
// Result(S, F) becomes S because transport throws domain and invoke failures instead of returning Err.
func clientInvokeReturnType(tm forstTypeMapper, returnTypeNode *ast.TypeNode) (string, error) {
	if returnTypeNode != nil && returnTypeNode.Ident == ast.TypeResult && len(returnTypeNode.TypeParams) >= 1 {
		return tm.GetTypeScriptType(&returnTypeNode.TypeParams[0])
	}
	return tm.GetTypeScriptType(returnTypeNode)
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
		tsType, err := clientInvokeReturnType(t.typeMapping, returnTypeNode)
		if err != nil {
			return nil, fmt.Errorf("failed to get TypeScript type for function return type %s: %w", returnTypeNode.Ident, err)
		}
		returnType = tsType
	} else if len(fn.ReturnTypes) > 0 {
		returnTypeNode := &fn.ReturnTypes[0]
		tsType, err := clientInvokeReturnType(t.typeMapping, returnTypeNode)
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
	if sig, exists := t.TypeChecker.Functions[fn.Ident.ID]; exists {
		domainMap := domainErrorsByName(t.Output.DomainErrors)
		signature.FailureType = FormatFunctionErrorUnion(sig, domainMap)
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
// Uses AsyncGenerator + local StreamingResult (emitted into types.d.ts) so row typing
// matches the inlined transport NDJSON envelope without @forst/sidecar.
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
		"export function %sStream(%s): AsyncGenerator<StreamingResult & { data?: %s }, void, undefined>;",
		fs.Name, paramStr, fs.StreamingRowType,
	)
}
