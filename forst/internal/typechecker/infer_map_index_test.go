package typechecker

import (
	"strings"
	"testing"

	"forst/internal/ast"
	"forst/internal/parser"

	"github.com/sirupsen/logrus"
)

func TestCheckTypes_mapIndex_singleValue(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	src := `package main
func main() {
	m := map[String]Int{ "a": 1, "b": 2 }
	x := m["a"]
	ensure x is Ok()
	println(string(x))
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestCheckTypes_mapCommaOk_rejected(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	src := `package main
func main() {
	m := map[String]Int{ "a": 1 }
	v, ok := m["a"]
	println(string(v))
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: map comma-ok is not supported")
	}
	if !strings.Contains(err.Error(), "comma-ok") {
		t.Fatalf("expected comma-ok diagnostic, got: %v", err)
	}
}

func TestCheckTypes_mapIndex_badKeyType(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	src := `package main
func main() {
	m := map[String]Int{ "a": 1 }
	println(string(m[1]))
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error for map index key type mismatch")
	}
	if !strings.Contains(err.Error(), "map index") {
		t.Fatalf("expected map index diagnostic, got: %v", err)
	}
}

func TestCheckTypes_twoLhs_sliceIndex_errors(t *testing.T) {
	t.Parallel()
	log := logrus.New()
	src := `package main
func main() {
	xs := [1, 2]
	a, b := xs[0]
	println(string(a))
}
`
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: slice index does not yield two values")
	}
}

func TestInferExpressionType_mapIndex_AST(t *testing.T) {
	tc := New(logrus.New(), false)
	m := astMapStringIntLiteral()
	idx := ast.IndexExpressionNode{
		Target: m,
		Index:  ast.StringLiteralNode{Value: "k"},
	}
	types, err := tc.inferExpressionType(idx)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || !types[0].IsResultType() || len(types[0].TypeParams) < 1 || types[0].TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("want Result(Int, Error), got %+v", types)
	}
}

func astMapStringIntLiteral() ast.MapLiteralNode {
	return ast.MapLiteralNode{
		Type: ast.TypeNode{
			Ident: ast.TypeMap,
			TypeParams: []ast.TypeNode{
				{Ident: ast.TypeString},
				{Ident: ast.TypeInt},
			},
		},
		Entries: []ast.MapEntryNode{
			{Key: ast.StringLiteralNode{Value: "k"}, Value: ast.IntLiteralNode{Value: 1}},
		},
	}
}

func TestCheckTypes_mapIndex_table(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		src     string
		wantErr bool
		errSub  string
	}{
		{
			name: "result_narrowed_then_string_ok",
			src: `package main
func main() {
	m := map[String]Int{ "a": 1 }
	x := m["a"]
	ensure x is Ok()
	println(string(x))
}`,
			wantErr: false,
		},
		{
			name: "string_on_raw_map_read_rejected",
			src: `package main
func main() {
	m := map[String]Int{ "a": 1 }
	println(string(m["a"]))
}`,
			wantErr: true,
			errSub:  "map lookup has type Result(V, Error)",
		},
		{
			name: "map_element_assignment",
			src: `package main
func main() {
	m := map[String]Int{ "a": 1 }
	m["a"] = 2
	println(string(m["a"]))
}`,
			wantErr: true, // still: m["a"] on RHS is Result until narrowed
			errSub:  "map lookup has type Result(V, Error)",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			log := logrus.New()
			p := parser.NewTestParser(tc.src, log)
			nodes, err := p.ParseFile()
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			chk := New(log, false)
			err = chk.CheckTypes(nodes)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected CheckTypes error")
				}
				if tc.errSub != "" && !strings.Contains(err.Error(), tc.errSub) {
					t.Fatalf("error %q should contain %q", err.Error(), tc.errSub)
				}
			} else {
				if err != nil {
					t.Fatalf("CheckTypes: %v", err)
				}
			}
		})
	}
}

func TestCheckTypes_mapAssign_updates_element_without_reading_result_twice(t *testing.T) {
	t.Parallel()
	// Only assignment to m[k]; no read of Result in println — use literals.
	src := `package main
func main() {
	m := map[String]Int{ "a": 1 }
	m["a"] = 2
	println("done")
}
`
	log := logrus.New()
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestCheckTypes_mapIndex_returnResult(t *testing.T) {
	t.Parallel()
	src := `package main
func lookup(): Result(Int, Error) {
	m := map[String]Int{ "a": 1 }
	return m["a"]
}
func main() {
	x := lookup()
	ensure x is Ok()
	println(string(x))
}
`
	log := logrus.New()
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestInferIndexExpressionAsAssignTarget_map_element(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	m := astMapStringIntLiteral()
	idx := ast.IndexExpressionNode{Target: m, Index: ast.StringLiteralNode{Value: "k"}}
	types, err := tc.inferIndexExpressionAsAssignTarget(idx)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].Ident != ast.TypeInt {
		t.Fatalf("assign target want Int, got %+v", types)
	}
}

func TestInferExpressionTypeWithExpected_mapIndex(t *testing.T) {
	t.Parallel()
	tc := New(logrus.New(), false)
	m := astMapStringIntLiteral()
	idx := ast.IndexExpressionNode{Target: m, Index: ast.StringLiteralNode{Value: "k"}}
	exp := ast.TypeNode{
		Ident: ast.TypeResult,
		TypeParams: []ast.TypeNode{
			ast.NewBuiltinType(ast.TypeInt),
			ast.NewBuiltinType(ast.TypeError),
		},
	}
	types, err := tc.inferExpressionTypeWithExpected(idx, &exp)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || !types[0].IsResultType() || types[0].TypeParams[0].Ident != ast.TypeInt {
		t.Fatalf("want Result(Int, Error), got %+v", types)
	}
}

func TestCheckTypes_nestedMapIndex_outerIsResult(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	inner := map[String]Int{ "b": 1 }
	m := map[String]map[String]Int{ "a": inner }
	x := m["a"]
	ensure x is Ok()
	println("ok")
}
`
	log := logrus.New()
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	if err := tc.CheckTypes(nodes); err != nil {
		t.Fatalf("CheckTypes: %v", err)
	}
}

func TestCheckTypes_chainedMapIndex_innerTargetNotMap(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
	inner := map[String]Int{ "b": 1 }
	m := map[String]map[String]Int{ "a": inner }
	println(string(m["a"]["b"]))
}
`
	log := logrus.New()
	p := parser.NewTestParser(src, log)
	nodes, err := p.ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tc := New(log, false)
	err = tc.CheckTypes(nodes)
	if err == nil {
		t.Fatal("expected CheckTypes error: cannot index Result without narrowing")
	}
	if !strings.Contains(err.Error(), "TYPE_RESULT") && !strings.Contains(err.Error(), "Result") {
		t.Fatalf("expected index-on-Result diagnostic, got: %v", err)
	}
}
