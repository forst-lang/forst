package lsp

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

func TestIdentifierPrefixAt(t *testing.T) {
	content := "  hello"
	got := identifierPrefixAt(content, LSPPosition{Line: 0, Character: 7})
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestLhsExpressionBeforeDot(t *testing.T) {
	if lhs := lhsExpressionBeforeDot("  foo."); lhs != "foo" {
		t.Fatalf("got %q", lhs)
	}
	if lhs := lhsExpressionBeforeDot("a.b."); lhs != "a.b" {
		t.Fatalf("got %q", lhs)
	}
	if lhs := lhsExpressionBeforeDot("  _ = s."); lhs != "s" {
		t.Fatalf("got %q", lhs)
	}
	if lhs := lhsExpressionBeforeDot("no dot"); lhs != "" {
		t.Fatalf("got %q", lhs)
	}
}

func TestListAtPosition_builtinMemberAfterDot(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "dot.ft")
	const src = `package main

func f() {
  var s: String = "a"
  _ = s.len
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// Cursor after `s.` on line `  _ = s.len`
	pos := LSPPosition{Line: 4, Character: 8}
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("analyze failed")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.TC == nil {
		t.Fatal("no typechecker")
	}
	tokIdx := tokenIndexAtLSPPosition(ctx.Tokens, pos)
	sn := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, ctx.TC)
	if sn == nil {
		t.Fatal("expected function scope node")
	}
	if err := ctx.TC.RestoreScope(sn); err != nil {
		t.Fatalf("RestoreScope: %v", err)
	}
	vn := ast.VariableNode{Ident: ast.Ident{ID: ast.Identifier("s")}}
	types, err := ctx.TC.InferExpressionTypeForCompletion(vn)
	if err != nil {
		t.Fatalf("infer s: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("no types for s")
	}

	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	found := make(map[string]bool)
	for _, it := range items {
		found[it.Label] = true
	}
	if !found["len"] {
		t.Fatalf("expected builtin String.len, got %#v", found)
	}
}

func TestListAtPosition_includesLocalVariable(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "locals.ft")
	const src = `package main

func main() {
  var x: Int = 1
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// After `var `, before `x` — identifier prefix empty
	pos := LSPPosition{Line: 3, Character: 6}
	items, _ := s.getCompletionsForPosition(uri, pos, nil)
	found := make(map[string]bool)
	for _, it := range items {
		found[it.Label] = true
	}
	if !found["x"] {
		t.Fatalf("expected local x in completions, got %#v", found)
	}
}

func TestListAtPosition_shapeFieldsAfterDot(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "shape_dot.ft")
	const src = `package main

type Row = {
  name: String
  count: Int
}

func f() {
  var r: Row
  _ = r.name
}
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	// After `r.` on line `  _ = r.name`
	pos := LSPPosition{Line: 9, Character: 8}
	items, _ := s.getCompletionsForPosition(uri, pos, &completionRequestContext{TriggerCharacter: "."})
	found := make(map[string]bool)
	for _, it := range items {
		found[it.Label] = true
	}
	if !found["name"] || !found["count"] {
		t.Fatalf("expected shape fields name and count, got %#v", found)
	}
}

func TestForstPackageNameFromContent(t *testing.T) {
	t.Parallel()
	if got := forstPackageNameFromContent("package main\n\nfunc f() {}\n"); got != "main" {
		t.Fatalf("got %q", got)
	}
	if got := forstPackageNameFromContent("// hi\npackage foo\n"); got != "foo" {
		t.Fatalf("got %q", got)
	}
}

func TestListAtPosition_crossBufferSamePackage(t *testing.T) {
	log := logrus.New()
	s := NewLSPServer("8080", log)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module t\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	aPath := filepath.Join(dir, "a.ft")
	bPath := filepath.Join(dir, "b.ft")
	const bsrc = "package main\n\nfunc OtherFunc() {\n}\n"
	const asrc = `package main

func main() {
  var x: Int = 1
}
`

	if err := os.WriteFile(bPath, []byte(bsrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aPath, []byte(asrc), 0o644); err != nil {
		t.Fatal(err)
	}
	uriA := mustFileURI(t, aPath)
	uriB := mustFileURI(t, bPath)
	s.documentMu.Lock()
	s.openDocuments[uriA] = asrc
	s.openDocuments[uriB] = bsrc
	s.documentMu.Unlock()

	pos := LSPPosition{Line: 3, Character: 2}
	items, incomplete := s.getCompletionsForPosition(uriA, pos, nil)
	if !incomplete {
		t.Fatal("expected isIncomplete true when multiple open .ft buffers")
	}
	labels := make([]string, 0, len(items))
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	if !slices.Contains(labels, "OtherFunc") {
		t.Fatalf("expected OtherFunc from open buffer b.ft, got %v", labels)
	}
}

func TestListAtPosition_insideCommentReturnsEmpty(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "comment.ft")
	const source = `package main

func main() {
  // typing here should not trigger completions
}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.documentMu.Lock()
	server.openDocuments[uri] = source
	server.documentMu.Unlock()

	// Position inside comment text.
	position := LSPPosition{Line: 3, Character: 10}
	items, incomplete := server.getCompletionsForPosition(uri, position, nil)
	if incomplete {
		t.Fatal("expected isIncomplete false in single-file comment case")
	}
	if len(items) != 0 {
		t.Fatalf("expected no completions inside comment, got %d: %+v", len(items), items)
	}
}

func TestListAtPosition_memberAfterDotUnknownLHSReturnsEmpty(t *testing.T) {
	log := logrus.New()
	server := NewLSPServer("8080", log)
	dir := t.TempDir()
	filePath := filepath.Join(dir, "dot_unknown.ft")
	const source = `package main

func main() {
  unknown.
}
`
	if err := os.WriteFile(filePath, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, filePath)
	server.documentMu.Lock()
	server.openDocuments[uri] = source
	server.documentMu.Unlock()

	position := LSPPosition{Line: 3, Character: 10}
	items, _ := server.getCompletionsForPosition(uri, position, &completionRequestContext{TriggerCharacter: "."})
	if len(items) != 0 {
		t.Fatalf("expected no member completions for unknown lhs, got %+v", items)
	}
}

func TestIfThenBraces_scanToOpeningThenBrace(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  if true {
    println("x")
  } else if false {
    println("y")
  } else {
    println("z")
  }
}
`
	toks := lexTokensForLSPHelperTest(src)
	ifIdx := -1
	for i, tk := range toks {
		if tk.Type == ast.TokenIf {
			ifIdx = i
			break
		}
	}
	if ifIdx < 0 {
		t.Fatal("no if token")
	}
	l, r := ifThenBraces(toks, ifIdx)
	if l < 0 || r < l {
		t.Fatalf("ifThenBraces: %d %d", l, r)
	}
	for i, tk := range toks {
		if tk.Type == ast.TokenElseIf {
			el, er := elseIfThenBraces(toks, i)
			if el < 0 || er < el {
				t.Fatalf("elseIfThenBraces: %d %d", el, er)
			}
			break
		}
		if tk.Type == ast.TokenElse && i+1 < len(toks) && toks[i+1].Type == ast.TokenIf {
			el, er := elseIfThenBraces(toks, i)
			if el < 0 || er < el {
				t.Fatalf("elseIfThenBraces (else+if): %d %d", el, er)
			}
			break
		}
	}
	src2 := `package main
func main() {
  if x := 1; x > 0 {
    println("a")
  }
}
`
	toks2 := lexTokensForLSPHelperTest(src2)
	ifIdx2 := -1
	for i, tk := range toks2 {
		if tk.Type == ast.TokenIf {
			ifIdx2 = i
			break
		}
	}
	l2, r2 := ifThenBraces(toks2, ifIdx2)
	if l2 < 0 || r2 < l2 {
		t.Fatalf("ifThenBraces short decl: %d %d", l2, r2)
	}
}

func TestIdentifierPrefixAt_grid(t *testing.T) {
	t.Parallel()
	src := "  foo_bar123  \n"
	lines := strings.Count(src, "\n")
	for line := 0; line <= lines; line++ {
		for ch := 0; ch < 20; ch++ {
			_ = identifierPrefixAt(src, LSPPosition{Line: line, Character: ch})
		}
	}
}

func TestLineUpToCursor_edgeLines(t *testing.T) {
	t.Parallel()
	src := "a\nbc\ndef\n"
	for line := -1; line < 5; line++ {
		for ch := -1; ch < 10; ch++ {
			_ = lineUpToCursor(src, LSPPosition{Line: line, Character: ch})
		}
	}
}

func TestTokenIndexAtLSPPosition_grid(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  x := 1
}
`
	toks := lexTokensForLSPHelperTest(src)
	for line := 0; line < 6; line++ {
		for ch := 0; ch < 40; ch++ {
			_ = tokenIndexAtLSPPosition(toks, LSPPosition{Line: line, Character: ch})
		}
	}
}

func TestInferZoneFromCursor_variants(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  println(x)
}
`
	toks := lexTokensForLSPHelperTest(src)
	reqDot := &completionRequestContext{TriggerCharacter: "."}
	reqNil := (*completionRequestContext)(nil)
	for line := 0; line < 6; line++ {
		for ch := 0; ch < 30; ch++ {
			pos := LSPPosition{Line: line, Character: ch}
			_ = inferCompletionZone(toks, pos, src, reqDot)
			_ = inferCompletionZone(toks, pos, src, reqNil)
		}
	}
}

func TestKeywordsForZone_all(t *testing.T) {
	t.Parallel()
	_ = keywordsForZone(zoneUnknown)
	_ = keywordsForZone(zoneTopLevel)
	_ = keywordsForZone(zoneInsideBlock)
	_ = keywordsForZone(zoneMemberAfterDot)
}

func TestMatchingRBraceAndSkipParens_scanIndices(t *testing.T) {
	t.Parallel()
	src := `package main
func f((a Int)) {}
func main() {
  { { } }
}
`
	toks := lexTokensForLSPHelperTest(src)
	for i := range toks {
		_ = matchingRBrace(toks, i)
		_ = skipBalancedParens(toks, i)
	}
}

func TestFindLBraceAfterFuncSignature_returnNamed(t *testing.T) {
	t.Parallel()
	src := `package main
func g() : Int {
  return 1
}
`
	toks := lexTokensForLSPHelperTest(src)
	idx := findFuncKeywordIndex(toks, "g")
	if idx < 0 {
		t.Fatal("no func g")
	}
	lb := findLBraceAfterFuncSignature(toks, idx)
	if lb < 0 {
		t.Fatal("expected lbrace")
	}
	if matchingRBrace(toks, lb) < 0 {
		t.Fatal("expected rbrace")
	}
}

func TestParamListParenRange_main(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
}
`
	toks := lexTokensForLSPHelperTest(src)
	fn := ast.FunctionNode{Ident: ast.Ident{ID: "main"}}
	o, c, ok := paramListParenRange(toks, fn)
	if !ok || o < 0 || c < o {
		t.Fatalf("param range: ok=%v %d %d", ok, o, c)
	}
}

func TestFindFirstToken_scan(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {}
`
	toks := lexTokensForLSPHelperTest(src)
	for from := 0; from < len(toks); from++ {
		for lim := from; lim <= len(toks)+2; lim++ {
			_ = findFirstToken(toks, from, lim, ast.TokenFunc)
		}
	}
}

func TestElseBlockBraces_and_ensureBlockBraces(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  if true {
  } else {
    ensure x is Min(1) {
    }
  }
}
`
	toks := lexTokensForLSPHelperTest(src)
	for i := range toks {
		if toks[i].Type == ast.TokenElse {
			l, r := elseBlockBraces(toks, i)
			if l < 0 || r < l {
				t.Fatalf("elseBlockBraces at %d: %d %d", i, l, r)
			}
			break
		}
	}
	for i := range toks {
		if toks[i].Type == ast.TokenEnsure {
			l, r := ensureBlockBraces(toks, i)
			if l < 0 || r < l {
				t.Fatalf("ensureBlockBraces at %d: %d %d", i, l, r)
			}
			break
		}
	}
}

func TestNetBraceDepthBetween_sweep(t *testing.T) {
	t.Parallel()
	src := `package main
func main() {
  { }
}
`
	toks := lexTokensForLSPHelperTest(src)
	n := len(toks)
	for a := 0; a < n; a++ {
		for b := a; b < n; b++ {
			_ = netBraceDepthBetween(toks, a, b)
		}
	}
}

func TestKeywordItemsDedupe(t *testing.T) {
	t.Parallel()
	a := completionItemsFromKeywords([]string{"func", "type", "func"}, "f")
	b := dedupeCompletionItems(append(a, a...))
	if len(b) < 1 {
		t.Fatal("expected items")
	}
}

func TestForstPackageNameFromContent_variants(t *testing.T) {
	t.Parallel()
	for _, s := range []string{
		"",
		"package main\n",
		"package foo\n",
		"  \n package bar \n",
	} {
		_ = forstPackageNameFromContent(s)
	}
}

func TestURIDisplayBasename_variants(t *testing.T) {
	t.Parallel()
	for _, u := range []string{
		"",
		"file:///tmp/x.ft",
		"file:///a/b/c/d.ft",
	} {
		_ = uriDisplayBasename(u)
	}
}
