package transformerts

import (
	"errors"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/typechecker"
)

func TestFunctionSignature_StreamTypesDeclaration_table(t *testing.T) {
	tests := []struct {
		name string
		sig  FunctionSignature
		want string
	}{
		{
			name: "empty row type returns empty declaration",
			sig: FunctionSignature{
				Name:             "StreamEvents",
				StreamingRowType: "",
			},
			want: "",
		},
		{
			name: "includes params and row type",
			sig: FunctionSignature{
				Name: "StreamEvents",
				Parameters: []Parameter{
					{Name: "limit", Type: "number"},
					{Name: "topic", Type: "string"},
				},
				StreamingRowType: "{ id: string }",
			},
			want: "export function StreamEventsStream(limit: number, topic: string): AsyncGenerator<import('@forst/sidecar').StreamingResult & { data?: { id: string } }, void, undefined>;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sig.StreamTypesDeclaration()
			if got != tt.want {
				t.Fatalf("StreamTypesDeclaration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypeScriptTransformer_transformFunction_table(t *testing.T) {
	makeFn := func(name string, returnTypes ...ast.TypeNode) ast.FunctionNode {
		return ast.FunctionNode{
			Ident: ast.Ident{ID: ast.Identifier(name)},
			Params: []ast.ParamNode{
				ast.SimpleParamNode{
					Ident: ast.Ident{ID: "count"},
					Type:  ast.NewBuiltinType(ast.TypeInt),
				},
			},
			ReturnTypes: returnTypes,
		}
	}

	tests := []struct {
		name                     string
		fn                       ast.FunctionNode
		tcReturnType             *ast.TypeNode
		enableStreamingClients   bool
		wantReturnType           string
		wantStreamingRowType     string
		wantStreamable           bool
		wantDefinitionFragments  []string
		wantStreamDeclIsNonEmpty bool
	}{
		{
			name:            "uses typechecker return type over ast return",
			fn:              makeFn("FromTypechecker", ast.NewBuiltinType(ast.TypeString)),
			tcReturnType:    ptrType(ast.NewBuiltinType(ast.TypeBool)),
			wantReturnType:  "boolean",
			wantStreamable:  false,
			wantStreamingRowType: "",
			wantDefinitionFragments: []string{
				`return client.invoke("pkg", "FromTypechecker", args);`,
			},
		},
		{
			name:            "falls back to ast return type",
			fn:              makeFn("FromAstOnly", ast.NewBuiltinType(ast.TypeString)),
			wantReturnType:  "string",
			wantStreamable:  false,
			wantStreamingRowType: "",
		},
		{
			name:            "defaults to unknown when no return type",
			fn:              makeFn("NoReturns"),
			wantReturnType:  "unknown",
			wantStreamable:  false,
			wantStreamingRowType: "",
		},
		{
			name:                   "sets streaming row type for chan returns",
			fn:                     makeFn("StreamRows", ast.NewChannelType(ast.NewBuiltinType(ast.TypeString))),
			enableStreamingClients: true,
			wantReturnType:         "AsyncIterable<string>",
			wantStreamable:         true,
			wantStreamingRowType:   "string",
			wantStreamDeclIsNonEmpty: true,
		},
		{
			name:                   "keeps streamable true but row type empty for unknown channel element",
			fn:                     makeFn("StreamUnknown", ast.NewChannelType(ast.TypeNode{Ident: ast.TypeImplicit})),
			enableStreamingClients: true,
			wantReturnType:         "AsyncIterable<unknown>",
			wantStreamable:         true,
			wantStreamingRowType:   "",
			wantStreamDeclIsNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typechecker.New(nil, false)
			if tt.tcReturnType != nil {
				tc.Functions[tt.fn.Ident.ID] = typechecker.FunctionSignature{
					Ident:       tt.fn.Ident,
					ReturnTypes: []ast.TypeNode{*tt.tcReturnType},
				}
			}

			tr := New(tc, nil)
			tr.GenerateStreamingClients = tt.enableStreamingClients
			tr.Output.SetPackageName("pkg")

			got, err := tr.transformFunction(tt.fn)
			if err != nil {
				t.Fatalf("transformFunction() error = %v", err)
			}

			if got.Signature.ReturnType != tt.wantReturnType {
				t.Fatalf("ReturnType = %q, want %q", got.Signature.ReturnType, tt.wantReturnType)
			}
			if got.Signature.Streamable != tt.wantStreamable {
				t.Fatalf("Streamable = %v, want %v", got.Signature.Streamable, tt.wantStreamable)
			}
			if got.Signature.StreamingRowType != tt.wantStreamingRowType {
				t.Fatalf("StreamingRowType = %q, want %q", got.Signature.StreamingRowType, tt.wantStreamingRowType)
			}
			if got.Signature.Parameters[0].Type != "number" {
				t.Fatalf("param type = %q, want number", got.Signature.Parameters[0].Type)
			}
			for _, frag := range tt.wantDefinitionFragments {
				if !strings.Contains(got.Definition, frag) {
					t.Fatalf("definition missing %q:\n%s", frag, got.Definition)
				}
			}

			streamDecl := got.Signature.StreamTypesDeclaration()
			if tt.wantStreamDeclIsNonEmpty && streamDecl == "" {
				t.Fatal("expected non-empty stream declaration")
			}
			if !tt.wantStreamDeclIsNonEmpty && streamDecl != "" {
				t.Fatalf("expected empty stream declaration, got %q", streamDecl)
			}
		})
	}
}

func ptrType(tn ast.TypeNode) *ast.TypeNode {
	return &tn
}

type alwaysErrMapper struct {
	err error
}

var errForcedMapping = errors.New("forced mapping error")

func (m *alwaysErrMapper) SetTypeChecker(*typechecker.TypeChecker) {}
func (m *alwaysErrMapper) AddUserType(string, string)              {}
func (m *alwaysErrMapper) shapeTypeFieldLines(ast.ShapeNode) ([]string, error) {
	return nil, nil
}
func (m *alwaysErrMapper) GetTypeScriptType(*ast.TypeNode) (string, error) {
	return "", m.err
}

func TestTypeScriptTransformer_transformFunction_returnTypeErrorPaths(t *testing.T) {
	tests := []struct {
		name         string
		fn           ast.FunctionNode
		tcReturnType *ast.TypeNode
		wantErrFrag  string
	}{
		{
			name: "typechecker return mapping error",
			fn: ast.FunctionNode{
				Ident: ast.Ident{ID: "BoomFromSig"},
			},
			tcReturnType: ptrType(ast.NewBuiltinType(ast.TypeInt)),
			wantErrFrag:  "failed to get TypeScript type for function return type",
		},
		{
			name: "ast return mapping error",
			fn: ast.FunctionNode{
				Ident:       ast.Ident{ID: "BoomFromAst"},
				ReturnTypes: []ast.TypeNode{ast.NewBuiltinType(ast.TypeInt)},
			},
			wantErrFrag: "failed to get TypeScript type for function return type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := typechecker.New(nil, false)
			if tt.tcReturnType != nil {
				tc.Functions[tt.fn.Ident.ID] = typechecker.FunctionSignature{
					Ident:       tt.fn.Ident,
					ReturnTypes: []ast.TypeNode{*tt.tcReturnType},
				}
			}

			tr := New(tc, nil)
			tr.typeMapping = &alwaysErrMapper{err: errForcedMapping}
			tr.Output.SetPackageName("pkg")

			_, err := tr.transformFunction(tt.fn)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErrFrag) {
				t.Fatalf("error = %v, want fragment %q", err, tt.wantErrFrag)
			}
		})
	}
}
