package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseFile_WithFunctions(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "basic function with parameter",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "abc123", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 10},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 14},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 17},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 19},
				{Type: ast.TokenReturn, Value: "return", Line: 2, Column: 4},
				{Type: ast.TokenIntLiteral, Value: "1", Line: 2, Column: 11},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if functionNode.GetIdent() != "abc123" {
					t.Errorf("Expected function name 'abc123', got %s", functionNode.Ident.ID)
				}
				if len(functionNode.Body) != 1 {
					t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}
			},
		},
		{
			name: "function with ensure statement",
			tokens: []ast.Token{
				{Type: ast.TokenFunc, Value: "func", Line: 1, Column: 1},
				{Type: ast.TokenIdentifier, Value: "main", Line: 1, Column: 6},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 9},
				{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 10},
				{Type: ast.TokenInt, Value: "int", Line: 1, Column: 14},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 17},
				{Type: ast.TokenLBrace, Value: "{", Line: 1, Column: 19},
				{Type: ast.TokenEnsure, Value: "ensure", Line: 2, Column: 4},
				{Type: ast.TokenIdentifier, Value: "x", Line: 2, Column: 11},
				{Type: ast.TokenIs, Value: "is", Line: 2, Column: 13},
				{Type: ast.TokenInt, Value: "Int", Line: 2, Column: 15},
				{Type: ast.TokenDot, Value: ".", Line: 2, Column: 18},
				{Type: ast.TokenIdentifier, Value: "Min", Line: 2, Column: 19},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 22},
				{Type: ast.TokenIntLiteral, Value: "0", Line: 2, Column: 23},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 24},
				{Type: ast.TokenOr, Value: "or", Line: 2, Column: 26},
				{Type: ast.TokenIdentifier, Value: "TooSmall", Line: 2, Column: 28},
				{Type: ast.TokenLParen, Value: "(", Line: 2, Column: 36},
				{Type: ast.TokenStringLiteral, Value: "x must be at least 0", Line: 2, Column: 37},
				{Type: ast.TokenRParen, Value: ")", Line: 2, Column: 51},
				{Type: ast.TokenRBrace, Value: "}", Line: 3, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 3, Column: 2},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				functionNode := assertNodeType[ast.FunctionNode](t, nodes[0], "ast.FunctionNode")
				if functionNode.GetIdent() != "main" {
					t.Errorf("Expected function name 'main', got %s", functionNode.GetIdent())
				}
				if len(functionNode.Body) != 1 {
					t.Errorf("Expected 1 statement in function body, got %d", len(functionNode.Body))
				}

				ensureNode := assertNodeType[ast.EnsureNode](t, functionNode.Body[0], "ast.EnsureNode")
				if ensureNode.Variable.GetIdent() != "x" {
					t.Errorf("Expected variable name 'x', got '%s'", ensureNode.Variable.GetIdent())
				}

				if len(ensureNode.Assertion.Constraints) != 1 {
					t.Errorf("Expected 1 constraint, got %d", len(ensureNode.Assertion.Constraints))
				}

				constraint := ensureNode.Assertion.Constraints[0]
				if constraint.Name != "Min" {
					t.Errorf("Expected constraint 'Min', got %s", constraint.Name)
				}

				if len(constraint.Args) != 1 {
					t.Errorf("Expected 1 argument, got %d", len(constraint.Args))
				}

				arg := constraint.Args[0]
				if arg.Value == nil {
					t.Fatal("Expected value argument, got nil")
				}

				value := *arg.Value
				if value.String() != "0" {
					t.Errorf("Expected value '0', got %s", value.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ast.SetupTestLogger(nil)
			p := setupParser(tt.tokens, logger)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			tt.validate(t, nodes)
		})
	}
}

func TestParseFunctionDefinition_AllowsKeywordNamesWithSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "error keyword as function name",
			src:  `func error(msg String) {}`,
			want: "error",
		},
		{
			name: "use keyword as function name",
			src:  `func use(x Int) {}`,
			want: "use",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewTestParser(tt.src, ast.SetupTestLogger(nil))
			fn := p.parseFunctionDefinition()
			if got := fn.GetIdent(); got != tt.want {
				t.Fatalf("function name = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseParameterType_ShapeAssertionBranches(t *testing.T) {
	t.Parallel()

	p := NewTestParser(`func f(x Amount.Min(1), y { id: Int }) {}`, ast.SetupTestLogger(nil))
	fn := p.parseFunctionDefinition()
	if len(fn.Params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(fn.Params))
	}

	x := fn.Params[0].(ast.SimpleParamNode)
	if x.Type.Ident != ast.TypeAssertion || x.Type.Assertion == nil {
		t.Fatalf("x parameter type = %+v", x.Type)
	}
	if x.Type.Assertion.BaseType == nil || *x.Type.Assertion.BaseType != "Amount" {
		t.Fatalf("x assertion base type = %v", x.Type.Assertion.BaseType)
	}

	y := fn.Params[1].(ast.SimpleParamNode)
	if y.Type.Ident != ast.TypeShape || y.Type.Assertion == nil {
		t.Fatalf("y parameter type = %+v", y.Type)
	}
	if len(y.Type.Assertion.Constraints) != 1 || y.Type.Assertion.Constraints[0].Name != "Match" {
		t.Fatalf("unexpected y assertion: %+v", y.Type.Assertion)
	}
}

func TestParseParameterType_Branches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, typ ast.TypeNode)
	}{
		{
			name:  "assertion type branch",
			input: `Amount.Min(1)`,
			check: func(t *testing.T, typ ast.TypeNode) {
				t.Helper()
				if typ.Ident != ast.TypeAssertion || typ.Assertion == nil {
					t.Fatalf("type = %+v", typ)
				}
				if typ.Assertion.BaseType == nil || *typ.Assertion.BaseType != "Amount" {
					t.Fatalf("assertion base type = %v", typ.Assertion.BaseType)
				}
			},
		},
		{
			name:  "shape literal type branch",
			input: `{ id: Int }`,
			check: func(t *testing.T, typ ast.TypeNode) {
				t.Helper()
				if typ.Ident != ast.TypeShape || typ.Assertion == nil {
					t.Fatalf("type = %+v", typ)
				}
				if len(typ.Assertion.Constraints) != 1 || typ.Assertion.Constraints[0].Name != "Match" {
					t.Fatalf("unexpected assertion: %+v", typ.Assertion)
				}
			},
		},
		{
			name:  "shape identifier branch",
			input: `Shape`,
			check: func(t *testing.T, typ ast.TypeNode) {
				t.Helper()
				if typ.Ident != "Shape" {
					t.Fatalf("type ident = %s", typ.Ident)
				}
			},
		},
		{
			name:  "plain type fallback branch",
			input: `String`,
			check: func(t *testing.T, typ ast.TypeNode) {
				t.Helper()
				if typ.Ident != ast.TypeString {
					t.Fatalf("type ident = %s", typ.Ident)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := NewTestParser(tt.input, ast.SetupTestLogger(nil))
			typ := p.parseParameterType()
			tt.check(t, typ)
		})
	}
}
