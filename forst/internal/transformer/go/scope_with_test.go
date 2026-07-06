package transformergo

import (
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"
	"forst/internal/typechecker"
)

const withHostSrc = `package demo

type Logger = {info(msg String)}

type NopLogger = {}

func (NopLogger) info(msg String) {}

func host() {
	with { Logger: &NopLogger{} } {
		use logger: Logger
		logger.info("hi")
	}
}
`

func parseWithHost(t *testing.T) ([]ast.Node, *typechecker.TypeChecker, ast.Node, ast.WithNode) {
	t.Helper()
	log := ast.SetupTestLogger(nil)
	p := parser.NewTestParser(withHostSrc, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	var withNode ast.Node
	var with ast.WithNode
	for _, n := range nodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "host" {
			continue
		}
		for _, stmt := range fn.Body {
			if w, ok := stmt.(ast.WithNode); ok {
				withNode = stmt
				with = w
				break
			}
		}
	}
	if withNode == nil {
		t.Fatal("with node not found")
	}
	return nodes, tc, withNode, with
}

func TestTransformWithStatements_parseNodeScope(t *testing.T) {
	t.Parallel()
	_, tc, withNode, with := parseWithHost(t)
	tr := New(tc, nil)

	stmts, err := tr.transformWithStatements(withNode, with)
	if err != nil {
		t.Fatalf("transformWithStatements: %v", err)
	}
	if len(stmts) == 0 {
		t.Fatal("expected non-empty with body stmts")
	}
}

func TestResolveWithScopeNode_reparseFindsTypecheckScope(t *testing.T) {
	t.Parallel()
	parseOnce := func() []ast.Node {
		log := ast.SetupTestLogger(nil)
		p := parser.NewTestParser(withHostSrc, log)
		nodes, err := p.ParseFile()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		return nodes
	}
	findWith := func(nodes []ast.Node) (ast.Node, ast.WithNode) {
		for _, n := range nodes {
			fn, ok := n.(ast.FunctionNode)
			if !ok || fn.Ident.ID != "host" {
				continue
			}
			for _, stmt := range fn.Body {
				if w, ok := stmt.(ast.WithNode); ok {
					return stmt, w
				}
			}
		}
		t.Fatal("with not found")
		return nil, ast.WithNode{}
	}

	typecheckNodes := parseOnce()
	tc := typechecker.New(nil, false)
	if err := tc.CheckTypes(typecheckNodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	transformNodes := parseOnce()
	withStmt, with := findWith(transformNodes)
	tr := New(tc, nil)
	tr.entryNodes = transformNodes

	resolved := tr.resolveWithScopeNode(transformNodes, with)
	if !tc.HasScopeForNode(resolved) {
		tcWith, _ := findWith(typecheckNodes)
		t.Fatalf("resolved=%s hasScope=%v typecheckWith=%s hasTcScope=%v", resolved, tc.HasScopeForNode(resolved), tcWith, tc.HasScopeForNode(tcWith))
	}
	if !tc.HasScopeForNode(withStmt) {
		_, err := tr.transformWithStatements(withStmt, with)
		if err != nil {
			t.Fatalf("transformWithStatements after resolve: %v", err)
		}
	}
}

func TestTransformWithStatements_reparseResolvesTypecheckScope(t *testing.T) {
	t.Parallel()
	log := ast.SetupTestLogger(nil)
	parseOnce := func() []ast.Node {
		p := parser.NewTestParser(withHostSrc, log)
		nodes, err := p.ParseFile()
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		return nodes
	}
	typecheckNodes := parseOnce()
	tc := typechecker.New(log, false)
	if err := tc.CheckTypes(typecheckNodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
	transformNodes := parseOnce()
	var withStmt ast.Node
	var with ast.WithNode
	for _, n := range transformNodes {
		fn, ok := n.(ast.FunctionNode)
		if !ok || fn.Ident.ID != "host" {
			continue
		}
		for _, stmt := range fn.Body {
			if w, ok := stmt.(ast.WithNode); ok {
				withStmt = stmt
				with = w
				break
			}
		}
	}
	tr := New(tc, nil)
	tr.entryNodes = transformNodes
	_, err := tr.transformWithStatements(withStmt, with)
	if err != nil {
		t.Fatalf("transformWithStatements after re-parse: %v", err)
	}
}
