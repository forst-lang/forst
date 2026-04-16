package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"forst/internal/ast"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

func TestTokenSliceIndex_pointerOrValueMatch(t *testing.T) {
	t.Parallel()
	tokens := []ast.Token{
		{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "a"},
		{Line: 1, Column: 3, Type: ast.TokenDot, Value: "."},
	}
	if i := tokenSliceIndex(tokens, &tokens[0]); i != 0 {
		t.Fatalf("pointer match: got %d", i)
	}
	alias := ast.Token{Line: 1, Column: 1, Type: ast.TokenIdentifier, Value: "a"}
	if i := tokenSliceIndex(tokens, &alias); i != 0 {
		t.Fatalf("value match: got %d", i)
	}
	if i := tokenSliceIndex(tokens, &ast.Token{Line: 9, Column: 9}); i != -1 {
		t.Fatalf("missing: got %d", i)
	}
}

func TestTypeDefHoverMarkdown_typeGuard(t *testing.T) {
	t.Parallel()
	v := ast.TypeGuardNode{Ident: ast.Identifier("Strong")}
	if md := typeDefHoverMarkdown(nil, nil, v); !strings.Contains(md, "Strong") || !strings.Contains(md, "is") || !strings.Contains(md, "Type guard") {
		t.Fatalf("value TypeGuardNode: got %q", md)
	}
	if md := typeDefHoverMarkdown(nil, nil, &v); !strings.Contains(md, "Strong") {
		t.Fatalf("*TypeGuardNode: got %q", md)
	}
}

func TestTypeDefHoverMarkdown_nominalError(t *testing.T) {
	t.Parallel()
	def := ast.TypeDefNode{
		Ident: "NotPositive",
		Expr: ast.TypeDefErrorExpr{
			Payload: ast.ShapeNode{
				Fields: map[string]ast.ShapeFieldNode{
					"message": {Type: &ast.TypeNode{Ident: ast.TypeString}},
				},
			},
		},
	}
	md := typeDefHoverMarkdown(nil, nil, def)
	if !strings.Contains(md, "Nominal error") || !strings.Contains(md, "error NotPositive") || !strings.Contains(md, "message") {
		t.Fatalf("hover: %q", md)
	}
}

func TestFindHoverForPosition_errorDotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module errhover\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "err_hover.ft")
	const src = `package main

import "fmt"

func checkConditions(): Error {
	return nil
}

func main() {
	err := checkConditions()
	ensure !err {
		fmt.Println(err.Error())
	}
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	var errTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type != ast.TokenIdentifier || tok.Value != "Error" {
			continue
		}
		// Second identifier "Error" in err.Error() — skip Println's line (no err. prefix in token stream).
		if i >= 2 && ctx.Tokens[i-1].Type == ast.TokenDot && ctx.Tokens[i-2].Value == "err" {
			errTok = tok
			break
		}
	}
	if errTok == nil {
		t.Fatal("could not find Error token for err.Error()")
	}
	// Lexer columns are 1-based; LSP positions are 0-based (match tokenAtLSPPosition).
	h := s.findHoverForPosition(uri, LSPPosition{Line: errTok.Line - 1, Character: errTok.Column - 1})
	if h == nil {
		t.Fatal("nil hover on Error in err.Error")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "error") || !strings.Contains(val, "Error()") {
		t.Fatalf("expected go error.Error hover, got %q", val)
	}
}

func TestFindHoverForPosition_errMoveDotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module errhover\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ft := filepath.Join(dir, "err_move_hover.ft")
	const src = `package main

import "errors"

func main() {
	errMove := errors.New("x")
	println(errMove.Error())
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	var errTok *ast.Token
	for i := range ctx.Tokens {
		tok := &ctx.Tokens[i]
		if tok.Type != ast.TokenIdentifier || tok.Value != "Error" {
			continue
		}
		if i >= 2 && ctx.Tokens[i-1].Type == ast.TokenDot && ctx.Tokens[i-2].Value == "errMove" {
			errTok = tok
			break
		}
	}
	if errTok == nil {
		t.Fatal("could not find Error token for errMove.Error()")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: errTok.Line - 1, Character: errTok.Column - 1})
	if h == nil {
		t.Fatal("nil hover on Error in errMove.Error")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "error") || !strings.Contains(val, "Error()") {
		t.Fatalf("expected go error.Error hover, got %q", val)
	}

	// Caret on '(' after "Error" should still show method hover (not LParen).
	var openAfterError *ast.Token
	for i := range ctx.Tokens {
		if i+1 >= len(ctx.Tokens) {
			continue
		}
		tok := &ctx.Tokens[i]
		if tok.Type != ast.TokenIdentifier || tok.Value != "Error" {
			continue
		}
		if i < 2 || ctx.Tokens[i-1].Type != ast.TokenDot || ctx.Tokens[i-2].Value != "errMove" {
			continue
		}
		nxt := &ctx.Tokens[i+1]
		if nxt.Type == ast.TokenLParen {
			openAfterError = nxt
			break
		}
	}
	if openAfterError == nil {
		t.Fatal("could not find '(' after errMove.Error")
	}
	h2 := s.findHoverForPosition(uri, LSPPosition{Line: openAfterError.Line - 1, Character: openAfterError.Column - 1})
	if h2 == nil {
		t.Fatal("nil hover on '(' after errMove.Error")
	}
	if v := h2.Contents.Value; !strings.Contains(v, "error") || !strings.Contains(v, "Error()") {
		t.Fatalf("expected go error.Error hover when cursor on '(', got %q", v)
	}
}

func TestFindHoverForPosition_typeGuardDeclarationName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "guard_hover.ft")
	const src = `package main

type P = String

is (p P) Strong {
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	line := "is (p P) Strong {"
	char := strings.Index(line, "Strong")
	if char < 0 {
		t.Fatal("fixture: no Strong")
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: 4, Character: char})
	if h == nil {
		t.Fatal("nil hover on type guard name")
	}
	if !strings.Contains(h.Contents.Value, "Strong") {
		t.Fatalf("hover should mention guard name: %q", h.Contents.Value)
	}
}

func TestHoverTextForToken_keyword(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenFunc, Value: "func"}
	s := hoverTextForToken(tc, nil, tok, nil)
	if !strings.Contains(s, "**`func`**") || !strings.Contains(s, "Declares") {
		t.Fatalf("got %q", s)
	}
}

func TestHoverTextForToken_builtinGuardAfterIs(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tokens := []ast.Token{
		{Type: ast.TokenIdentifier, Value: "x", Line: 1, Column: 1},
		{Type: ast.TokenIs, Value: "is", Line: 1, Column: 3},
		{Type: ast.TokenIdentifier, Value: "Min", Line: 1, Column: 6},
	}
	tok := &tokens[2]
	if s := hoverTextForToken(tc, tokens, tok, nil); !strings.Contains(s, "Min") || !strings.Contains(s, "guard") {
		t.Fatalf("got %q", s)
	}
}

func TestHoverTextForToken_builtinTypeNameIdentifier(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenIdentifier, Value: "Tuple", Line: 1, Column: 1}
	if s := hoverTextForToken(tc, nil, tok, nil); !strings.Contains(s, "Tuple") || !strings.Contains(s, "built-in type") {
		t.Fatalf("got %q", s)
	}
}

func TestHoverTextForToken_nilKeyword(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenNil, Value: "nil", Line: 1, Column: 1}
	if s := hoverTextForToken(tc, nil, tok, nil); !strings.Contains(s, "nil") || !strings.Contains(s, "zero value") {
		t.Fatalf("got %q", s)
	}
}

func TestHoverTextForToken_stringLiteralNonImportReturnsEmpty(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenStringLiteral, Value: `"hello"`}
	if s := hoverTextForToken(tc, []ast.Token{{Type: ast.TokenStringLiteral, Value: `"hello"`}}, tok, nil); s != "" {
		t.Fatalf("expected no hover for non-import string, got %q", s)
	}
}

func TestHoverTextForToken_intLiteralReturnsEmpty(t *testing.T) {
	t.Parallel()
	tc := typechecker.New(logrus.New(), false)
	tok := &ast.Token{Type: ast.TokenIntLiteral, Value: "42"}
	if s := hoverTextForToken(tc, nil, tok, nil); s != "" {
		t.Fatalf("got %q", s)
	}
}

func TestLexicalHoverMarkdown_keywordAndIdentifier(t *testing.T) {
	t.Parallel()
	if s := lexicalHoverMarkdown(&ast.Token{Type: ast.TokenFunc, Value: "func"}); !strings.Contains(s, "**`func`**") {
		t.Fatalf("keyword: got %q", s)
	}
	id := &ast.Token{Type: ast.TokenIdentifier, Value: "foo"}
	if s := lexicalHoverMarkdown(id); !strings.Contains(s, "`foo`") || !strings.Contains(s, "parses") {
		t.Fatalf("identifier: got %q", s)
	}
	if s := lexicalHoverMarkdown(&ast.Token{Type: ast.TokenIntLiteral, Value: "1"}); s != "" {
		t.Fatalf("literal: got %q", s)
	}
}

func TestFindHoverForPosition_parseError_keywordHover(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "bad_hover.ft")
	// Same top-level rejection as analyze_test (parser error, tokens still present).
	const src = "package main\n\nunexpected\n"
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr == nil {
		t.Fatal("expected parse error for test fixture")
	}

	// Line 0: `package` keyword
	hPkg := s.findHoverForPosition(uri, LSPPosition{Line: 0, Character: 2})
	if hPkg == nil || !strings.Contains(hPkg.Contents.Value, "**`package`**") {
		if hPkg == nil {
			t.Fatal("expected keyword hover on package when parse fails")
		}
		t.Fatalf("package hover: got %q", hPkg.Contents.Value)
	}
	// Line 2: `unexpected` — lexical identifier hover
	h := s.findHoverForPosition(uri, LSPPosition{Line: 2, Character: 2})
	if h == nil || !strings.Contains(h.Contents.Value, "`unexpected`") {
		if h == nil {
			t.Fatal("expected hover when parse fails")
		}
		t.Fatalf("identifier hover: got %q", h.Contents.Value)
	}
	_ = ctx
}

func TestHandleWorkspaceSymbol_filtersQuery(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	ftPath := filepath.Join(dir, "ws.ft")
	const src = `package main

func fooBar(): Int { return 0 }
func other(): Int { return 1 }
`
	if err := os.WriteFile(ftPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ftPath)
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	resp := s.handleWorkspaceSymbol(LSPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "workspace/symbol",
		Params:  mustJSONParams(t, map[string]interface{}{"query": "bar"}),
	})
	if resp.Error != nil {
		t.Fatalf("error: %+v", resp.Error)
	}
	syms, ok := resp.Result.([]LspSymbolInformation)
	if !ok {
		t.Fatalf("result type %T", resp.Result)
	}
	if len(syms) != 1 || syms[0].Name != "fooBar" {
		t.Fatalf("got %#v", syms)
	}
}

func TestFindHoverForPosition_ifBranchNarrowingShowsRefinedVariableType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "narrow_hover.ft")
	const src = `package main

type MyStr = String

func f(): String {
	x := "hi"
	if x is MyStr {
		return x
	}
	return ""
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(src, "\n")
	var hoverLine, charOffset int
	for i, line := range lines {
		if strings.Contains(line, "return x") && strings.Contains(line, "return") {
			hoverLine = i
			charOffset = strings.Index(line, "x")
			break
		}
	}
	if charOffset < 0 {
		t.Fatal("could not find inner return x")
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	linePrefix := []rune(lines[hoverLine][:charOffset])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(string(linePrefix))})
	if h == nil {
		t.Fatal("nil hover on narrowed x")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "MyStr") && !strings.Contains(val, "String") {
		t.Fatalf("hover should mention type; got %q", val)
	}
	if !strings.Contains(val, "MyStr()") || !strings.Contains(val, "String.") {
		t.Fatalf("hover should use dotted predicate chain (e.g. String.MyStr()); got %q", val)
	}
}

func TestFindHoverForPosition_ensureBlockNarrowingShowsRefinedVariableType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "ensure_narrow_hover.ft")
	const src = `package main

type MyStr = String

func main() {
	x := "hi"
	ensure x is MyStr {
		y := x
		return
	}
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(src, "\n")
	var hoverLine, charOffset int
	for i, line := range lines {
		if strings.Contains(line, "y := x") {
			hoverLine = i
			charOffset = strings.Index(line, "x")
			break
		}
	}
	if charOffset < 0 {
		t.Fatal("could not find x in y := x")
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	linePrefix := []rune(lines[hoverLine][:charOffset])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(string(linePrefix))})
	if h == nil {
		t.Fatal("nil hover on x after ensure narrowing")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "MyStr") && !strings.Contains(val, "String") {
		t.Fatalf("hover should mention narrowed type; got %q", val)
	}
	_ = ctx
}

func TestFindHoverForPosition_typeGuardSuccessiveEnsureAccumulatesPredicateDisplay(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "tg_two_ensure_hover.ft")
	const src = `package main

is (x: String) G {
	ensure x is Min(1)
	ensure x is Max(10)
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(src, "\n")
	var hoverLine, charOffset int
	for i, line := range lines {
		if strings.Contains(line, "ensure x is Max") {
			hoverLine = i
			charOffset = strings.Index(line, "x")
			break
		}
	}
	if charOffset < 0 {
		t.Fatal("could not find subject x on second ensure")
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	linePrefix := []rune(lines[hoverLine][:charOffset])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(string(linePrefix))})
	if h == nil {
		t.Fatal("nil hover on x")
	}
	val := h.Contents.Value
	// Subject on this line is typed before this ensure's narrowing; only prior ensures apply.
	if !strings.Contains(val, "Min(1)") {
		t.Fatalf("hover should show prior ensure predicate; got %q", val)
	}
	if strings.Contains(val, "Max(10)") {
		t.Fatalf("hover on second ensure subject must not include this line's Max(10); got %q", val)
	}
	_ = ctx
}

func TestFindHoverForPosition_typeGuardFieldPathSecondEnsureShowsPriorPredicate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "tg_field_ensure_hover.ft")
	const src = `package main

type GameState = {
	cells: []String,
}

is (g GameState) ValidBoard() {
	ensure g.cells is Min(9)
	ensure g.cells is Max(9)
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(src, "\n")
	var hoverLine, charOffset int
	for i, line := range lines {
		if strings.Contains(line, "ensure g.cells is Max") {
			hoverLine = i
			charOffset = strings.Index(line, "cells")
			break
		}
	}
	if charOffset < 0 {
		t.Fatal("could not find cells on second ensure")
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	linePrefix := []rune(lines[hoverLine][:charOffset])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(string(linePrefix))})
	if h == nil {
		t.Fatal("nil hover on cells")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "Array(String)") {
		t.Fatalf("hover should mention slice type; got %q", val)
	}
	if !strings.Contains(val, "Min(9)") {
		t.Fatalf("hover on second ensure subject should show prior Min(9); got %q", val)
	}
	if strings.Contains(val, "Max(9)") {
		t.Fatalf("hover on second ensure subject must not include this line's Max(9); got %q", val)
	}
	_ = ctx
}

func TestFindHoverForPosition_functionParamFieldPathAfterEnsureTypeGuard(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "fn_param_field_ensure_tg_hover.ft")
	const src = `package main

type GameState = {
	cells: []String,
	status: String,
}

is (g GameState) ValidBoard() {
	ensure g.cells is Min(9)
	ensure g.cells is Max(9)
}

type MoveRequest = {
	state: GameState,
	row:   Int,
	col:   Int,
}

func ApplyMove(req MoveRequest): Result(Int, Error) {
	ensure req.state is ValidBoard()
	if req.state.status != "playing" {
		return 0
	}
	return 1
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(src, "\n")
	var hoverLine, charOffset int
	for i, line := range lines {
		if strings.Contains(line, `if req.state.status`) {
			hoverLine = i
			charOffset = strings.Index(line, "state")
			break
		}
	}
	if charOffset < 0 {
		t.Fatal("could not find req.state on if line")
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	linePrefix := []rune(lines[hoverLine][:charOffset])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(string(linePrefix))})
	if h == nil {
		t.Fatal("nil hover on state in req.state")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "GameState") {
		t.Fatalf("hover should mention GameState; got %q", val)
	}
	if !strings.Contains(val, "ValidBoard") {
		t.Fatalf("hover after ensure should include ValidBoard; got %q", val)
	}
	_ = ctx
}

func TestFindHoverForPosition_ensureNarrowingListsTypeGuardComments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "ensure_guard_doc_hover.ft")
	const src = `package main

type Password = String

// Strength: at least 12 characters.
is (password Password) Strong {
  ensure password is Min(12)
}

func f(): String {
  password: Password = "123456789012"
  if password is Strong() {
    return password
  }
  return ""
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(src, "\n")
	var hoverLine, charOffset int
	for i, line := range lines {
		if strings.Contains(line, "return password") && strings.Contains(line, "if") == false {
			hoverLine = i
			charOffset = strings.Index(line, "password")
			break
		}
	}
	if charOffset < 0 {
		t.Fatal("could not find password in then-branch return")
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil {
		t.Fatal("expected analyzed document")
	}
	if ctx.ParseErr != nil {
		t.Fatalf("parse: %v", ctx.ParseErr)
	}
	if ctx.CheckErr != nil {
		t.Fatalf("check: %v", ctx.CheckErr)
	}

	linePrefix := []rune(lines[hoverLine][:charOffset])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(string(linePrefix))})
	if h == nil {
		t.Fatal("nil hover on password in narrowed branch")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "Strong") && !strings.Contains(val, "Password") {
		t.Fatalf("hover should mention refined type; got %q", val)
	}
	if !strings.Contains(val, "Strength:") || !strings.Contains(val, "12 characters") {
		t.Fatalf("hover should include type guard comment block; got %q", val)
	}
	if iDoc, iFence := strings.Index(val, "Strength:"), strings.Index(val, "```"); iDoc < 0 || iFence < 0 || iDoc >= iFence {
		t.Fatalf("doc should appear above the forst code block; got %q", val)
	}
	_ = ctx
}

func TestHandleHover_invalidParamsReturnsParseError(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleHover(LSPRequest{
		JSONRPC: "2.0",
		ID:      42,
		Params:  json.RawMessage(`not-json`),
	})
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error for invalid params")
	}
	if resp.Error.Code != -32700 {
		t.Fatalf("want code -32700, got %d msg %q", resp.Error.Code, resp.Error.Message)
	}
	if resp.ID != 42 {
		t.Fatalf("echo id: got %v", resp.ID)
	}
}

func TestHandleTextDocumentList_invalidParamsReturnsParseError(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	resp := s.handleCompletion(LSPRequest{
		JSONRPC: "2.0",
		ID:      7,
		Params:  json.RawMessage(`{`),
	})
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error for invalid params")
	}
	if resp.Error.Code != -32700 {
		t.Fatalf("want code -32700, got %d", resp.Error.Code)
	}
}

func TestLexicalHoverMarkdown_trueFalseNil(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tok  ast.TokenIdent
		want string
	}{
		{ast.TokenTrue, "true"},
		{ast.TokenFalse, "false"},
		{ast.TokenNil, "nil"},
	}
	for _, tc := range cases {
		tok := ast.Token{Type: tc.tok, Value: tc.want}
		s := lexicalHoverMarkdown(&tok)
		if s == "" || !strings.Contains(s, tc.want) {
			t.Fatalf("%v: got %q", tc.tok, s)
		}
	}
}

func TestFindHover_crossFileTypeLeadingComment(t *testing.T) {
	t.Parallel()
	s := NewLSPServer("8080", logrus.New())
	dir := t.TempDir()
	aPath := filepath.Join(dir, "widget.ft")
	bPath := filepath.Join(dir, "use.ft")
	const srcA = `package main

// Widget wraps an identifier
type Widget = {
  id: Int
}
`
	const srcB = `package main

func wid(w: Widget): Int {
  return w.id
}
`
	if err := os.WriteFile(aPath, []byte(srcA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bPath, []byte(srcB), 0o644); err != nil {
		t.Fatal(err)
	}
	uriA := mustFileURI(t, aPath)
	uriB := mustFileURI(t, bPath)
	s.documentMu.Lock()
	s.openDocuments[uriA] = srcA
	s.openDocuments[uriB] = srcB
	s.documentMu.Unlock()

	pos := lspPositionOfIdentifier(srcB, "Widget")
	h := s.findHoverForPosition(uriB, pos)
	if h == nil {
		t.Fatal("expected hover on Widget in peer file")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "Widget wraps an identifier") {
		t.Fatalf("expected merged // doc from defining file, got %q", val)
	}
	if !strings.Contains(val, "type Widget") && !strings.Contains(val, "Widget") {
		t.Fatalf("expected type definition in hover: %q", val)
	}
}

func TestFindHoverForPosition_noTokenReturnsNil(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "short.ft")
	const src = "package main\n"
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()
	// Line 10 has no tokens in this one-line file.
	if h := s.findHoverForPosition(uri, LSPPosition{Line: 10, Character: 0}); h != nil {
		t.Fatalf("expected nil hover when no token at position, got %#v", h)
	}
}

func TestFindHover_userTypeNameShadowsBuiltinTupleDoc(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "shadow_tuple.ft")
	const src = `package main

type Tuple = Int

func f(): Tuple {
  return 0
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	line := "func f(): Tuple {"
	ix := strings.Index(line, "Tuple")
	if ix < 0 {
		t.Fatal("fixture")
	}
	li := 0
	for i, l := range strings.Split(src, "\n") {
		if strings.Contains(l, "func f():") {
			li = i
			break
		}
	}
	h := s.findHoverForPosition(uri, LSPPosition{Line: li, Character: ix})
	if h == nil {
		t.Fatal("nil hover on Tuple return type")
	}
	val := h.Contents.Value
	// User-defined alias must win over built-in Tuple FFI hover.
	if !strings.Contains(val, "type Tuple") {
		t.Fatalf("expected user type hover, got %q", val)
	}
	if strings.Contains(val, "FFI") || strings.Contains(val, "multi-value") {
		t.Fatalf("did not expect built-in Tuple blurb, got %q", val)
	}
}

func TestFindHover_variableHoverIncludesBuiltinGuardDocWhenNoSourceComment(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "guard_fallback_hover.ft")
	const src = `package main

func f(s: String): Void {
  ensure s is Min(3)
  _ = s
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	lines := strings.Split(src, "\n")
	var hoverLine, charOff int
	for i, line := range lines {
		if strings.Contains(line, "_ = s") {
			hoverLine = i
			charOff = strings.Index(line, "s")
			break
		}
	}
	if charOff < 0 {
		t.Fatal("could not find s in _ = s")
	}
	prefix := string([]rune(lines[hoverLine])[:charOff])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(prefix)})
	if h == nil {
		t.Fatal("nil hover on s after ensure")
	}
	val := h.Contents.Value
	// Built-in Min guard doc (hoverdoc) should appear when there is no // above the guard.
	if !strings.Contains(val, "Min") || !strings.Contains(val, "minimum") {
		t.Fatalf("expected built-in Min guard documentation in hover, got %q", val)
	}
}

func TestFindHover_guardIdentifierShowsBaseQualifiedTitle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ft := filepath.Join(dir, "guard_qual_hover.ft")
	const src = `package main

func f(): Void {
	n := 1
	ensure n is GreaterThan(0)
}
`
	if err := os.WriteFile(ft, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	uri := mustFileURI(t, ft)
	s := NewLSPServer("8080", logrus.New())
	s.documentMu.Lock()
	s.openDocuments[uri] = src
	s.documentMu.Unlock()

	lines := strings.Split(src, "\n")
	var hoverLine, charOff int
	for i, line := range lines {
		if ix := strings.Index(line, "GreaterThan"); ix >= 0 {
			hoverLine = i
			charOff = ix
			break
		}
	}
	if charOff < 0 {
		t.Fatal("could not find GreaterThan")
	}
	prefix := string([]rune(lines[hoverLine])[:charOff])
	h := s.findHoverForPosition(uri, LSPPosition{Line: hoverLine, Character: utf8.RuneCountInString(prefix)})
	if h == nil {
		t.Fatal("nil hover on GreaterThan")
	}
	val := h.Contents.Value
	if !strings.Contains(val, "Int.GreaterThan") || !strings.Contains(val, "strictly above") {
		t.Fatalf("expected Int.GreaterThan qualified guard title and body, got %q", val)
	}
}
