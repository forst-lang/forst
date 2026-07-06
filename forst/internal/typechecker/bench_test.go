package typechecker

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/testutil"
)

func benchExampleSource(b *testing.B, rel string) []byte {
	b.Helper()
	path := testutil.ExamplePath(b, rel)
	src, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("read example %s: %v", rel, err)
	}
	return src
}

func benchParseNodes(b *testing.B, src []byte) []ast.Node {
	b.Helper()
	return testutil.ParseSourceForBench(b, src, "bench.ft")
}

func benchTypeChecker(b *testing.B) *TypeChecker {
	b.Helper()
	return NewTypeCheckerForBench(b)
}

func BenchmarkCheckTypes_basic(b *testing.B) {
	src := benchExampleSource(b, "basic.ft")
	nodes := benchParseNodes(b, src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := benchTypeChecker(b)
		if err := tc.CheckTypes(nodes); err != nil {
			b.Fatalf("CheckTypes: %v", err)
		}
	}
}

func BenchmarkRestoreScope_functionBody(b *testing.B) {
	var body []ast.Node
	for i := 0; i < 200; i++ {
		body = append(body, ast.AssignmentNode{
			LValues: []ast.ExpressionNode{ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier(fmt.Sprintf("v%d", i))}}},
			RValues: []ast.ExpressionNode{ast.IntLiteralNode{Value: int64(i)}},
		})
	}
	fn := ast.FunctionNode{
		Ident: ast.Ident{ID: "benchFn"},
		Body:  body,
	}
	nodes := []ast.Node{fn}
	tc := benchTypeChecker(b)
	if err := tc.CollectTypes(nodes); err != nil {
		b.Fatalf("CollectTypes: %v", err)
	}
	scopeNode := nodes[0]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := tc.RestoreScope(scopeNode); err != nil {
			b.Fatalf("RestoreScope: %v", err)
		}
	}
}

func BenchmarkInferNodeTypes_paramList(b *testing.B) {
	const paramCount = 20
	params := make([]ast.ParamNode, paramCount)
	paramNodes := make([]ast.Node, paramCount)
	for i := 0; i < paramCount; i++ {
		p := ast.SimpleParamNode{
			Ident: ast.Ident{ID: ast.Identifier(fmt.Sprintf("p%d", i))},
			Type:  ast.TypeNode{Ident: ast.TypeInt},
		}
		params[i] = p
		paramNodes[i] = p
	}
	fn := ast.FunctionNode{
		Ident:  ast.Ident{ID: "manyParams"},
		Params: params,
		Body:   []ast.Node{},
	}
	tc := benchTypeChecker(b)
	if err := tc.CollectTypes([]ast.Node{fn}); err != nil {
		b.Fatalf("CollectTypes: %v", err)
	}
	if err := tc.RestoreScope(fn); err != nil {
		b.Fatalf("RestoreScope: %v", err)
	}
	for _, param := range fn.Params {
		switch typedParam := param.(type) {
		case ast.SimpleParamNode:
			tc.scopeStack.currentScope().RegisterSymbol(
				typedParam.Ident.ID,
				[]ast.TypeNode{typedParam.Type},
				SymbolVariable)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := tc.inferNodeTypes(paramNodes, fn); err != nil {
			b.Fatalf("inferNodeTypes: %v", err)
		}
	}
}

func BenchmarkInferExpression_repeatedSubexpr(b *testing.B) {
	src := []byte(`package bench

func F(): Int {
	a := (1 + 2) * (1 + 2)
	return a
}
`)
	nodes := benchParseNodes(b, src)
	tc := benchTypeChecker(b)
	if err := tc.CheckTypes(nodes); err != nil {
		b.Fatalf("CheckTypes: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc2 := benchTypeChecker(b)
		if err := tc2.CheckTypes(nodes); err != nil {
			b.Fatalf("CheckTypes: %v", err)
		}
	}
}

func syntheticLargeFunctionSource(stmtCount int) string {
	var b strings.Builder
	b.WriteString("package bench\n\nfunc LargeFn(): Void {\n")
	for i := 0; i < stmtCount; i++ {
		fmt.Fprintf(&b, "\tx%d := %d\n", i, i)
	}
	b.WriteString("}\n")
	return b.String()
}

func BenchmarkCheckTypes_largeFunction(b *testing.B) {
	src := []byte(syntheticLargeFunctionSource(100))
	nodes := benchParseNodes(b, src)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := benchTypeChecker(b)
		if err := tc.CheckTypes(nodes); err != nil {
			b.Fatalf("CheckTypes: %v", err)
		}
	}
}
