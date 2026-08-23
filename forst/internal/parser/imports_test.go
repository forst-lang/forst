package parser

import (
	"forst/internal/ast"
	"testing"
)

func TestParseImport(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, importNode ast.ImportNode)
	}{
		{
			name: "simple import",
			tokens: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 6},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "fmt" {
					t.Errorf("Expected path 'fmt', got %s", importNode.Path)
				}
				if importNode.Alias != nil {
					t.Errorf("Expected no alias, got %s", importNode.Alias.ID)
				}
			},
		},
		{
			name: "import with alias",
			tokens: []ast.Token{
				{Type: ast.TokenIdentifier, Value: "f", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 3},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 8},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "fmt" {
					t.Errorf("Expected path 'fmt', got %s", importNode.Path)
				}
				if importNode.Alias == nil {
					t.Fatal("Expected alias, got nil")
				}
				if importNode.Alias.ID != "f" {
					t.Errorf("Expected alias 'f', got %s", importNode.Alias.ID)
				}
			},
		},
		{
			name: "dot import",
			tokens: []ast.Token{
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"strings"`, Line: 1, Column: 3},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 12},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "strings" {
					t.Errorf("Expected path 'strings', got %s", importNode.Path)
				}
				if importNode.Alias == nil {
					t.Fatal("Expected alias for dot import")
				}
				if importNode.Alias.ID != "." {
					t.Errorf("Expected alias '.', got %s", importNode.Alias.ID)
				}
				if importNode.SideEffectOnly {
					t.Error("dot import must not be side-effect-only")
				}
			},
		},
		{
			name: "import with complex path",
			tokens: []ast.Token{
				{Type: ast.TokenStringLiteral, Value: `"github.com/example/pkg"`, Line: 1, Column: 1},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 26},
			},
			validate: func(t *testing.T, importNode ast.ImportNode) {
				if importNode.Path != "github.com/example/pkg" {
					t.Errorf("Expected path 'github.com/example/pkg', got %s", importNode.Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ast.SetupTestLogger(nil)
			p := setupParser(tt.tokens, logger)
			importNode := p.parseImport(false)
			tt.validate(t, importNode)
		})
	}
}

func TestParseImportGroup(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 2},
		{Type: ast.TokenIdentifier, Value: "f", Line: 1, Column: 7},
		{Type: ast.TokenStringLiteral, Value: `"os"`, Line: 1, Column: 9},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 13},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 14},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	importGroup := p.parseImportGroup()

	if len(importGroup.Imports) != 2 {
		t.Fatalf("Expected 2 imports, got %d", len(importGroup.Imports))
	}

	// First import (fmt)
	if importGroup.Imports[0].Path != "fmt" {
		t.Errorf("Expected first import path 'fmt', got %s", importGroup.Imports[0].Path)
	}
	if importGroup.Imports[0].Alias != nil {
		t.Errorf("Expected first import to have no alias, got %s", importGroup.Imports[0].Alias.ID)
	}

	// Second import (os with alias f)
	if importGroup.Imports[1].Path != "os" {
		t.Errorf("Expected second import path 'os', got %s", importGroup.Imports[1].Path)
	}
	if importGroup.Imports[1].Alias == nil {
		t.Fatal("Expected second import to have alias")
	}
	if importGroup.Imports[1].Alias.ID != "f" {
		t.Errorf("Expected second import alias 'f', got %s", importGroup.Imports[1].Alias.ID)
	}
}

func TestParseImports(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []ast.Token
		validate func(t *testing.T, nodes []ast.Node)
	}{
		{
			name: "single import",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 8},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 13},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				importNode := assertNodeType[ast.ImportNode](t, nodes[0], "ast.ImportNode")
				if importNode.Path != "fmt" {
					t.Errorf("Expected import path 'fmt', got %s", importNode.Path)
				}
			},
		},
		{
			name: "import group",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 8},
				{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 9},
				{Type: ast.TokenStringLiteral, Value: `"os"`, Line: 1, Column: 14},
				{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 18},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 19},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				importGroup := assertNodeType[ast.ImportGroupNode](t, nodes[0], "ast.ImportGroupNode")
				if len(importGroup.Imports) != 2 {
					t.Fatalf("Expected 2 imports in group, got %d", len(importGroup.Imports))
				}
				if importGroup.Imports[0].Path != "fmt" {
					t.Errorf("Expected first import path 'fmt', got %s", importGroup.Imports[0].Path)
				}
				if importGroup.Imports[1].Path != "os" {
					t.Errorf("Expected second import path 'os', got %s", importGroup.Imports[1].Path)
				}
			},
		},
		{
			name: "dot import single line",
			tokens: []ast.Token{
				{Type: ast.TokenImport, Value: "import", Line: 1, Column: 1},
				{Type: ast.TokenDot, Value: ".", Line: 1, Column: 8},
				{Type: ast.TokenStringLiteral, Value: `"strings"`, Line: 1, Column: 10},
				{Type: ast.TokenEOF, Value: "", Line: 1, Column: 19},
			},
			validate: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				importNode := assertNodeType[ast.ImportNode](t, nodes[0], "ast.ImportNode")
				if importNode.Path != "strings" {
					t.Errorf("Expected path 'strings', got %s", importNode.Path)
				}
				if importNode.Alias == nil || importNode.Alias.ID != "." {
					t.Fatalf("Expected dot alias, got %+v", importNode.Alias)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := ast.SetupTestLogger(nil)
			p := setupParser(tt.tokens, logger)
			nodes := p.parseImports()
			tt.validate(t, nodes)
		})
	}
}

func TestParseImport_importStarAsFrom_rejected(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenStar, Value: "*", Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "as", Line: 1, Column: 3},
		{Type: ast.TokenIdentifier, Value: "payment", Line: 1, Column: 6},
		{Type: ast.TokenIdentifier, Value: "from", Line: 1, Column: 14},
		{Type: ast.TokenStringLiteral, Value: `"./legacy/payment"`, Line: 1, Column: 19},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 39},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected parse error for import * as … from")
		}
	}()
	_ = p.parseImport(false)
}

func TestParseImport_importJSPostfix_setsBridgeOptIn(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenStringLiteral, Value: `"./legacy/payment"`, Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "js", Line: 1, Column: 21},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 24},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	importNode := p.parseImport(false)

	if importNode.Path != "./legacy/payment" {
		t.Errorf("expected path ./legacy/payment, got %q", importNode.Path)
	}
	if importNode.Alias != nil {
		t.Errorf("expected no alias for postfix JS import, got %v", importNode.Alias)
	}
	if !importNode.BridgeOptIn {
		t.Error("expected BridgeOptIn true")
	}
	if importNode.BridgeOptInSource != "import_js" {
		t.Errorf("expected BridgeOptInSource import_js, got %q", importNode.BridgeOptInSource)
	}
}

func TestParseImport_importJSPostfixWithAlias_setsBridgeOptIn(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "checkout", Line: 1, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"./legacy/api/checkout"`, Line: 1, Column: 10},
		{Type: ast.TokenIdentifier, Value: "js", Line: 1, Column: 34},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 37},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	importNode := p.parseImport(false)

	if importNode.Path != "./legacy/api/checkout" {
		t.Errorf("expected path ./legacy/api/checkout, got %q", importNode.Path)
	}
	if importNode.Alias == nil || importNode.Alias.ID != "checkout" {
		t.Fatalf("expected alias checkout, got %+v", importNode.Alias)
	}
	if !importNode.BridgeOptIn {
		t.Error("expected BridgeOptIn true")
	}
	if importNode.BridgeOptInSource != "import_js" {
		t.Errorf("expected BridgeOptInSource import_js, got %q", importNode.BridgeOptInSource)
	}
}

func TestParseImport_goAliasNode_notBridgeOptIn(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "node", Line: 1, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"fmt"`, Line: 1, Column: 6},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 11},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	importNode := p.parseImport(false)

	if importNode.Path != "fmt" {
		t.Errorf("expected path fmt, got %q", importNode.Path)
	}
	if importNode.Alias == nil || importNode.Alias.ID != "node" {
		t.Fatalf("expected alias node, got %+v", importNode.Alias)
	}
	if importNode.BridgeOptIn {
		t.Error("expected BridgeOptIn false for Go alias import node \"fmt\"")
	}
}

func TestParseImport_importJSPostfix_groupedImport(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenLParen, Value: "(", Line: 1, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"strconv"`, Line: 1, Column: 2},
		{Type: ast.TokenStringLiteral, Value: `"./a.ts"`, Line: 1, Column: 11},
		{Type: ast.TokenIdentifier, Value: "js", Line: 1, Column: 20},
		{Type: ast.TokenIdentifier, Value: "checkout", Line: 1, Column: 23},
		{Type: ast.TokenStringLiteral, Value: `"./b.ts"`, Line: 1, Column: 32},
		{Type: ast.TokenIdentifier, Value: "js", Line: 1, Column: 41},
		{Type: ast.TokenRParen, Value: ")", Line: 1, Column: 44},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 45},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	importGroup := p.parseImportGroup()

	if len(importGroup.Imports) != 3 {
		t.Fatalf("expected 3 imports, got %d", len(importGroup.Imports))
	}
	if importGroup.Imports[0].BridgeOptIn {
		t.Error("strconv import should not have BridgeOptIn")
	}
	if !importGroup.Imports[1].BridgeOptIn || importGroup.Imports[1].BridgeOptInSource != "import_js" {
		t.Errorf("second import: expected import_js opt-in, got optIn=%v source=%q",
			importGroup.Imports[1].BridgeOptIn, importGroup.Imports[1].BridgeOptInSource)
	}
	if importGroup.Imports[1].Alias != nil {
		t.Errorf("second import: expected no alias, got %v", importGroup.Imports[1].Alias)
	}
	if !importGroup.Imports[2].BridgeOptIn || importGroup.Imports[2].Alias == nil || importGroup.Imports[2].Alias.ID != "checkout" {
		t.Errorf("third import: expected postfix js with alias checkout, got %+v", importGroup.Imports[2])
	}
}

func TestParseImport_importNodePostfix_rejected(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenStringLiteral, Value: `"./legacy/payment"`, Line: 1, Column: 1},
		{Type: ast.TokenIdentifier, Value: "node", Line: 1, Column: 21},
		{Type: ast.TokenEOF, Value: "", Line: 1, Column: 26},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected parse error for removed postfix node marker")
		}
	}()
	_ = p.parseImport(false)
}

func TestParseImport_commentDirective_doesNotApplyOptIn(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenComment, Value: "// forst:node", Line: 1, Column: 1},
		{Type: ast.TokenImport, Value: "import", Line: 2, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"./legacy/payment"`, Line: 2, Column: 8},
		{Type: ast.TokenEOF, Value: "", Line: 3, Column: 1},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (comment + import), got %d", len(nodes))
	}

	firstImport := assertNodeType[ast.ImportNode](t, nodes[1], "ast.ImportNode")
	if firstImport.BridgeOptIn {
		t.Error("expected forst:node directive to be ignored; use import \"./path\" js")
	}
}

func TestParseImport_commentDirective_fileScope_ignored(t *testing.T) {
	tokens := []ast.Token{
		{Type: ast.TokenComment, Value: "// forst:node file", Line: 1, Column: 1},
		{Type: ast.TokenImport, Value: "import", Line: 2, Column: 1},
		{Type: ast.TokenStringLiteral, Value: `"./legacy/payment"`, Line: 2, Column: 8},
		{Type: ast.TokenEOF, Value: "", Line: 3, Column: 1},
	}

	logger := ast.SetupTestLogger(nil)
	p := setupParser(tokens, logger)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	firstImport := assertNodeType[ast.ImportNode](t, nodes[1], "ast.ImportNode")
	if firstImport.BridgeOptIn {
		t.Error("expected forst:node file directive to be ignored; use import \"./path\" node")
	}
}
