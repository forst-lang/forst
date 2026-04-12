package lsp

import (
	"forst/internal/ast"
	"forst/internal/hasher"
	"forst/internal/typechecker"
)

// definingTokenForLocalBinding resolves same-file definition for parameters and locals
// after top-level navigable symbols have been ruled out.
func definingTokenForLocalBinding(ctx *forstDocumentContext, tokIdx int, tok *ast.Token) *ast.Token {
	if ctx == nil || ctx.TC == nil || tok == nil || tok.Type != ast.TokenIdentifier {
		return nil
	}
	tc := ctx.TC
	if err := tc.RestoreScope(nil); err != nil {
		return nil
	}
	scopeNode := findInnermostScopeNode(ctx.Nodes, ctx.Tokens, tokIdx, tc)
	if scopeNode != nil {
		if err := tc.RestoreScope(scopeNode); err != nil {
			return nil
		}
	}
	sym, ok := tc.CurrentScope().LookupVariable(ast.Identifier(tok.Value))
	if !ok || sym.Scope == nil {
		return nil
	}
	want := sym.Scope
	astNode := findASTNodeForScope(tc, ctx.Nodes, want)
	if astNode == nil {
		return nil
	}
	return definingTokenFromASTNode(astNode, tc, ctx.Nodes, ctx.Tokens, tok.Value)
}

// findASTNodeForScope walks the parsed tree and returns the AST node whose restored
// scope equals want (pointer identity).
func findASTNodeForScope(tc *typechecker.TypeChecker, nodes []ast.Node, want *typechecker.Scope) ast.Node {
	for _, n := range nodes {
		_ = tc.RestoreScope(nil)
		if r := visitNodeForScope(tc, n, want); r != nil {
			return r
		}
	}
	return nil
}

func visitNodeForScope(tc *typechecker.TypeChecker, n ast.Node, want *typechecker.Scope) ast.Node {
	if err := tc.RestoreScope(n); err == nil && tc.CurrentScope() == want {
		return n
	}
	switch v := n.(type) {
	case ast.FunctionNode:
		for _, st := range v.Body {
			if r := visitStmtForScope(tc, st, want); r != nil {
				return r
			}
		}
	case *ast.FunctionNode:
		return visitNodeForScope(tc, *v, want)
	case ast.TypeGuardNode:
		for _, st := range v.Body {
			if r := visitStmtForScope(tc, st, want); r != nil {
				return r
			}
		}
	case *ast.TypeGuardNode:
		for _, st := range v.Body {
			if r := visitStmtForScope(tc, st, want); r != nil {
				return r
			}
		}
	}
	return nil
}

func visitStmtForScope(tc *typechecker.TypeChecker, st ast.Node, want *typechecker.Scope) ast.Node {
	switch v := st.(type) {
	case ast.IfNode:
		return visitIfChainForScope(tc, v, want)
	case *ast.IfNode:
		return visitIfChainForScope(tc, *v, want)
	case ast.ForNode:
		if err := tc.RestoreScope(v); err == nil && tc.CurrentScope() == want {
			return v
		}
		for _, inner := range v.Body {
			if r := visitStmtForScope(tc, inner, want); r != nil {
				return r
			}
		}
	case *ast.ForNode:
		return visitStmtForScope(tc, *v, want)
	case ast.EnsureNode:
		if err := tc.RestoreScope(v); err == nil && tc.CurrentScope() == want {
			return v
		}
		if v.Block != nil {
			if err := tc.RestoreScope(v.Block); err == nil && tc.CurrentScope() == want {
				return v.Block
			}
			for _, inner := range v.Block.Body {
				if r := visitStmtForScope(tc, inner, want); r != nil {
					return r
				}
			}
		}
	case *ast.EnsureNode:
		return visitStmtForScope(tc, *v, want)
	}
	return nil
}

func visitIfChainForScope(tc *typechecker.TypeChecker, ifn ast.IfNode, want *typechecker.Scope) ast.Node {
	if err := tc.RestoreScope(&ifn); err == nil && tc.CurrentScope() == want {
		return &ifn
	}
	for _, inner := range ifn.Body {
		if r := visitStmtForScope(tc, inner, want); r != nil {
			return r
		}
	}
	for i := range ifn.ElseIfs {
		ei := ifn.ElseIfs[i]
		if err := tc.RestoreScope(&ei); err == nil && tc.CurrentScope() == want {
			return &ei
		}
		for _, inner := range ei.Body {
			if r := visitStmtForScope(tc, inner, want); r != nil {
				return r
			}
		}
	}
	if ifn.Else != nil {
		if err := tc.RestoreScope(ifn.Else); err == nil && tc.CurrentScope() == want {
			return ifn.Else
		}
		for _, inner := range ifn.Else.Body {
			if r := visitStmtForScope(tc, inner, want); r != nil {
				return r
			}
		}
	}
	return nil
}

func definingTokenFromASTNode(n ast.Node, tc *typechecker.TypeChecker, fileNodes []ast.Node, tokens []ast.Token, name string) *ast.Token {
	switch v := n.(type) {
	case ast.FunctionNode:
		if t := findParamIdentTokenForFunction(tokens, v, name); t != nil {
			return t
		}
		lb, rb := functionBodyBraces(tokens, v)
		if lb >= 0 && rb >= 0 {
			return findFirstShortDeclIdentToken(tokens, lb+1, rb, name)
		}
	case *ast.TypeGuardNode:
		return definingTokenForTypeGuardParams(tokens, *v, name)
	case ast.TypeGuardNode:
		return definingTokenForTypeGuardParams(tokens, v, name)
	case *ast.ForNode:
		return definingTokenForFor(tc, fileNodes, tokens, v, name)
	case ast.ForNode:
		return definingTokenForFor(tc, fileNodes, tokens, &v, name)
	case *ast.IfNode:
		return definingTokenForIf(tc, fileNodes, tokens, *v, name)
	case ast.IfNode:
		return definingTokenForIf(tc, fileNodes, tokens, v, name)
	case *ast.ElseIfNode:
		return findShortDeclTokenInBlockBody(fileNodes, tokens, v.Body, name, tc.Hasher)
	case ast.ElseIfNode:
		return findShortDeclTokenInBlockBody(fileNodes, tokens, v.Body, name, tc.Hasher)
	case *ast.ElseBlockNode:
		return findShortDeclTokenInBlockBody(fileNodes, tokens, v.Body, name, tc.Hasher)
	case ast.ElseBlockNode:
		return findShortDeclTokenInBlockBody(fileNodes, tokens, v.Body, name, tc.Hasher)
	case *ast.EnsureBlockNode:
		return findShortDeclTokenInBlockBody(fileNodes, tokens, v.Body, name, tc.Hasher)
	case ast.EnsureBlockNode:
		return findShortDeclTokenInBlockBody(fileNodes, tokens, v.Body, name, tc.Hasher)
	}
	return nil
}

// findShortDeclTokenInBlockBody finds the k-th `name :=` in the program matching the first
// short declaration in body (DFS), using structural hashes to disambiguate.
func findShortDeclTokenInBlockBody(fileNodes []ast.Node, tokens []ast.Token, body []ast.Node, name string, h *hasher.StructuralHasher) *ast.Token {
	assign := findFirstShortDeclAssignmentInBody(body, name)
	if assign == nil {
		return nil
	}
	want, err := h.HashNode(*assign)
	if err != nil {
		return nil
	}
	k := globalShortDeclOrdinalForAssignment(fileNodes, name, want, h)
	if k < 0 {
		return nil
	}
	return nthShortDeclIdentToken(tokens, name, k)
}

func findFirstShortDeclAssignmentInBody(body []ast.Node, name string) *ast.AssignmentNode {
	var walk func([]ast.Node) *ast.AssignmentNode
	walk = func(stmts []ast.Node) *ast.AssignmentNode {
		for _, s := range stmts {
			if a, ok := s.(ast.AssignmentNode); ok && a.IsShort {
				for _, lv := range a.LValues {
					vn, ok := lv.(ast.VariableNode)
					if !ok || string(vn.Ident.ID) != name {
						continue
					}
					copyA := a
					return &copyA
				}
			}
			switch st := s.(type) {
			case ast.IfNode:
				if r := walk(st.Body); r != nil {
					return r
				}
				for i := range st.ElseIfs {
					if r := walk(st.ElseIfs[i].Body); r != nil {
						return r
					}
				}
				if st.Else != nil {
					if r := walk(st.Else.Body); r != nil {
						return r
					}
				}
			case *ast.IfNode:
				if r := walk(st.Body); r != nil {
					return r
				}
				for i := range st.ElseIfs {
					if r := walk(st.ElseIfs[i].Body); r != nil {
						return r
					}
				}
				if st.Else != nil {
					if r := walk(st.Else.Body); r != nil {
						return r
					}
				}
			case ast.ForNode:
				if r := walk(st.Body); r != nil {
					return r
				}
			case *ast.ForNode:
				if r := walk(st.Body); r != nil {
					return r
				}
			case ast.EnsureNode:
				if st.Block != nil {
					if r := walk(st.Block.Body); r != nil {
						return r
					}
				}
			case *ast.EnsureNode:
				if st.Block != nil {
					if r := walk(st.Block.Body); r != nil {
						return r
					}
				}
			}
		}
		return nil
	}
	return walk(body)
}

func globalShortDeclOrdinalForAssignment(fileNodes []ast.Node, name string, want hasher.NodeHash, h *hasher.StructuralHasher) int {
	idx := 0
	var walk func(ast.Node) bool
	walk = func(node ast.Node) bool {
		switch v := node.(type) {
		case ast.AssignmentNode:
			if !v.IsShort {
				return false
			}
			for _, lv := range v.LValues {
				vn, ok := lv.(ast.VariableNode)
				if !ok || string(vn.Ident.ID) != name {
					continue
				}
				got, err := h.HashNode(v)
				if err != nil {
					return false
				}
				if got == want {
					return true
				}
				idx++
			}
		case ast.FunctionNode:
			for _, st := range v.Body {
				if walk(st) {
					return true
				}
			}
		case ast.TypeGuardNode:
			for _, st := range v.Body {
				if walk(st) {
					return true
				}
			}
		case *ast.TypeGuardNode:
			return walk(*v)
		case ast.IfNode:
			if v.Init != nil && walk(v.Init) {
				return true
			}
			for _, st := range v.Body {
				if walk(st) {
					return true
				}
			}
			for i := range v.ElseIfs {
				for _, st := range v.ElseIfs[i].Body {
					if walk(st) {
						return true
					}
				}
			}
			if v.Else != nil {
				for _, st := range v.Else.Body {
					if walk(st) {
						return true
					}
				}
			}
		case *ast.IfNode:
			return walk(*v)
		case ast.ForNode:
			if v.Init != nil && walk(v.Init) {
				return true
			}
			for _, st := range v.Body {
				if walk(st) {
					return true
				}
			}
		case *ast.ForNode:
			return walk(*v)
		case ast.EnsureNode:
			if v.Block != nil {
				for _, st := range v.Block.Body {
					if walk(st) {
						return true
					}
				}
			}
		case *ast.EnsureNode:
			return walk(*v)
		}
		return false
	}
	for _, top := range fileNodes {
		if walk(top) {
			return idx
		}
	}
	return -1
}

func nthShortDeclIdentToken(tokens []ast.Token, name string, k int) *ast.Token {
	if k < 0 {
		return nil
	}
	seen := 0
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Type != ast.TokenIdentifier || tokens[i].Value != name {
			continue
		}
		if tokens[i+1].Type != ast.TokenColonEquals {
			continue
		}
		if seen == k {
			return &tokens[i]
		}
		seen++
	}
	return nil
}

func findParamIdentTokenForFunction(tokens []ast.Token, fn ast.FunctionNode, name string) *ast.Token {
	idx := findFuncKeywordIndex(tokens, string(fn.Ident.ID))
	if idx < 0 {
		return nil
	}
	j := idx + 2
	if j >= len(tokens) || tokens[j].Type != ast.TokenLParen {
		return nil
	}
	closeParen := skipBalancedParens(tokens, j)
	if closeParen < 0 {
		return nil
	}
	for _, p := range fn.Params {
		sp, ok := p.(ast.SimpleParamNode)
		if !ok || string(sp.Ident.ID) != name {
			continue
		}
		for k := j + 1; k < closeParen; k++ {
			if tokens[k].Type == ast.TokenIdentifier && tokens[k].Value == name {
				// Go-style parameters: `x Int` (no colon). Legacy `x: Type` also places the name token first.
				return &tokens[k]
			}
		}
	}
	return nil
}

func definingTokenForTypeGuardParams(tokens []ast.Token, g ast.TypeGuardNode, name string) *ast.Token {
	guardName := string(g.Ident)
	for i := 0; i+1 < len(tokens); i++ {
		if braceDepthAtIndex(tokens, i) != 0 {
			continue
		}
		if tokens[i].Type != ast.TokenIs || tokens[i+1].Type != ast.TokenLParen {
			continue
		}
		open := i + 1
		closeParen := skipBalancedParens(tokens, open)
		if closeParen < 0 {
			continue
		}
		k := closeParen + 1
		for k < len(tokens) && tokens[k].Type == ast.TokenComment {
			k++
		}
		if k >= len(tokens) || tokens[k].Type != ast.TokenIdentifier || tokens[k].Value != guardName {
			continue
		}
		for p := open + 1; p < closeParen; p++ {
			if tokens[p].Type == ast.TokenIdentifier && tokens[p].Value == name {
				if p+1 < len(tokens) && tokens[p+1].Type == ast.TokenColon {
					return &tokens[p]
				}
			}
		}
	}
	return nil
}

func definingTokenForFor(tc *typechecker.TypeChecker, fileNodes []ast.Node, tokens []ast.Token, forn *ast.ForNode, name string) *ast.Token {
	nth := forNodeOccurrenceIndex(fileNodes, forn, tc.Hasher)
	if nth < 0 {
		return nil
	}
	fk := nthTopLevelForKeyword(tokens, nth)
	if fk < 0 {
		return nil
	}
	if forn.IsRange {
		if forn.RangeKey != nil && string(forn.RangeKey.ID) == name {
			return findIdentBetweenForAndRange(tokens, fk, name)
		}
		if forn.RangeValue != nil && string(forn.RangeValue.ID) == name {
			return findIdentBetweenForAndRange(tokens, fk, name)
		}
	}
	if forn.Init != nil {
		if a, ok := forn.Init.(ast.AssignmentNode); ok && a.IsShort {
			for _, lv := range a.LValues {
				vn, vok := lv.(ast.VariableNode)
				if vok && string(vn.Ident.ID) == name {
					return findIdentInForThreeClauseHeader(tokens, fk, name)
				}
			}
		}
	}
	lb, rb := forStmtBodyBraces(tokens, fk)
	if lb >= 0 && rb >= 0 {
		return findFirstShortDeclIdentToken(tokens, lb+1, rb, name)
	}
	return nil
}

func forNodeOccurrenceIndex(nodes []ast.Node, target *ast.ForNode, h *hasher.StructuralHasher) int {
	want, err := h.HashNode(*target)
	if err != nil {
		return -1
	}
	n := 0
	found := false
	var walk func(ast.Node)
	walk = func(node ast.Node) {
		if found {
			return
		}
		switch v := node.(type) {
		case ast.FunctionNode:
			for _, st := range v.Body {
				walk(st)
			}
		case ast.TypeGuardNode:
			for _, st := range v.Body {
				walk(st)
			}
		case *ast.TypeGuardNode:
			walk(*v)
		case ast.ForNode:
			got, e := h.HashNode(v)
			if e == nil && got == want {
				found = true
				return
			}
			n++
			for _, st := range v.Body {
				walk(st)
			}
		case *ast.ForNode:
			walk(*v)
		case ast.IfNode:
			for _, st := range v.Body {
				walk(st)
			}
			for i := range v.ElseIfs {
				for _, st := range v.ElseIfs[i].Body {
					walk(st)
				}
			}
			if v.Else != nil {
				for _, st := range v.Else.Body {
					walk(st)
				}
			}
		case *ast.IfNode:
			walk(*v)
		case ast.EnsureNode:
			if v.Block != nil {
				for _, st := range v.Block.Body {
					walk(st)
				}
			}
		case *ast.EnsureNode:
			walk(*v)
		}
	}
	for _, top := range nodes {
		walk(top)
		if found {
			break
		}
	}
	if !found {
		return -1
	}
	return n
}

func nthTopLevelForKeyword(tokens []ast.Token, n int) int {
	if n < 0 {
		return -1
	}
	seen := 0
	for i := range tokens {
		if tokens[i].Type == ast.TokenFor && braceDepthAtIndex(tokens, i) == 0 {
			if seen == n {
				return i
			}
			seen++
		}
	}
	return -1
}

func findIdentBetweenForAndRange(tokens []ast.Token, forIdx int, name string) *ast.Token {
	j := forIdx + 1
	for j < len(tokens) && tokens[j].Type != ast.TokenRange {
		if tokens[j].Type == ast.TokenIdentifier && tokens[j].Value == name {
			return &tokens[j]
		}
		j++
	}
	return nil
}

func findIdentInForThreeClauseHeader(tokens []ast.Token, forIdx int, name string) *ast.Token {
	j := forIdx + 1
	for j < len(tokens) && tokens[j].Type != ast.TokenLBrace {
		if tokens[j].Type == ast.TokenIdentifier && tokens[j].Value == name {
			return &tokens[j]
		}
		j++
	}
	return nil
}

func bodyLBraceAfterForHeader(tokens []ast.Token, forIdx int) int {
	depth := 0
	for j := forIdx + 1; j < len(tokens); j++ {
		switch tokens[j].Type {
		case ast.TokenLParen:
			depth++
		case ast.TokenRParen:
			if depth > 0 {
				depth--
			}
		case ast.TokenLBrace:
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

func forStmtBodyBraces(tokens []ast.Token, forKeywordIdx int) (l, r int) {
	lb := bodyLBraceAfterForHeader(tokens, forKeywordIdx)
	if lb < 0 {
		return -1, -1
	}
	rb := matchingRBrace(tokens, lb)
	return lb, rb
}

// findIfKeywordTokenIndexForIfNode returns the token index of the `if` keyword that starts
// the given IfNode (including if statements nested inside function or type-guard bodies).
func findIfKeywordTokenIndexForIfNode(tokens []ast.Token, fileNodes []ast.Node, ifn ast.IfNode, h *hasher.StructuralHasher) int {
	want, err := h.HashNode(ifn)
	if err != nil {
		return -1
	}
	var walk func(ast.Node) int
	walk = func(node ast.Node) int {
		switch v := node.(type) {
		case ast.FunctionNode:
			lb, rb := functionBodyBraces(tokens, v)
			if lb < 0 {
				return -1
			}
			if ix := searchStmtListForIfKeyword(tokens, v.Body, lb, rb, want, h); ix >= 0 {
				return ix
			}
		case *ast.FunctionNode:
			return walk(*v)
		case ast.TypeGuardNode:
			lb, rb := typeGuardBodyBraces(tokens, string(v.Ident))
			if lb < 0 {
				return -1
			}
			if ix := searchStmtListForIfKeyword(tokens, v.Body, lb, rb, want, h); ix >= 0 {
				return ix
			}
		case *ast.TypeGuardNode:
			return walk(*v)
		}
		return -1
	}
	for _, top := range fileNodes {
		if ix := walk(top); ix >= 0 {
			return ix
		}
	}
	return -1
}

func searchStmtListForIfKeyword(tokens []ast.Token, body []ast.Node, blockL, blockR int, want hasher.NodeHash, h *hasher.StructuralHasher) int {
	ifIdx := 0
	for _, st := range body {
		switch v := st.(type) {
		case ast.IfNode:
			idx := findNthBlockLevelKeyword(tokens, blockL, blockR, ast.TokenIf, ifIdx)
			ifIdx++
			if idx < 0 {
				continue
			}
			got, err := h.HashNode(v)
			if err == nil && got == want {
				return idx
			}
			l, r := ifThenBraces(tokens, idx)
			if l >= 0 && r > l {
				if inner := searchStmtListForIfKeyword(tokens, v.Body, l, r, want, h); inner >= 0 {
					return inner
				}
			}
		case *ast.IfNode:
			idx := findNthBlockLevelKeyword(tokens, blockL, blockR, ast.TokenIf, ifIdx)
			ifIdx++
			if idx < 0 {
				continue
			}
			got, err := h.HashNode(*v)
			if err == nil && got == want {
				return idx
			}
			l, r := ifThenBraces(tokens, idx)
			if l >= 0 && r > l {
				if inner := searchStmtListForIfKeyword(tokens, v.Body, l, r, want, h); inner >= 0 {
					return inner
				}
			}
		}
	}
	return -1
}

func definingTokenForIf(tc *typechecker.TypeChecker, fileNodes []ast.Node, tokens []ast.Token, ifn ast.IfNode, name string) *ast.Token {
	if ifn.Init != nil {
		if a, ok := ifn.Init.(ast.AssignmentNode); ok && a.IsShort {
			for _, lv := range a.LValues {
				vn, vok := lv.(ast.VariableNode)
				if vok && string(vn.Ident.ID) == name {
					return findIdentInsideIfInitParens(tokens, fileNodes, ifn, tc.Hasher, name)
				}
			}
		}
	}
	ifIdx := findIfKeywordTokenIndexForIfNode(tokens, fileNodes, ifn, tc.Hasher)
	if ifIdx < 0 {
		return nil
	}
	l, r := ifThenBraces(tokens, ifIdx)
	if l < 0 {
		return nil
	}
	return findFirstShortDeclIdentToken(tokens, l+1, r, name)
}

func findIdentInsideIfInitParens(tokens []ast.Token, fileNodes []ast.Node, ifn ast.IfNode, h *hasher.StructuralHasher, name string) *ast.Token {
	ifIdx := findIfKeywordTokenIndexForIfNode(tokens, fileNodes, ifn, h)
	if ifIdx < 0 {
		return nil
	}
	if ifIdx+1 >= len(tokens) || tokens[ifIdx+1].Type != ast.TokenLParen {
		return nil
	}
	closeIdx := skipBalancedParens(tokens, ifIdx+1)
	if closeIdx < 0 {
		return nil
	}
	for k := ifIdx + 2; k < closeIdx; k++ {
		if tokens[k].Type == ast.TokenIdentifier && tokens[k].Value == name {
			return &tokens[k]
		}
	}
	return nil
}

func findFirstShortDeclIdentToken(tokens []ast.Token, from, to int, name string) *ast.Token {
	if from < 0 {
		from = 0
	}
	if to > len(tokens) {
		to = len(tokens)
	}
	for i := from; i+1 < to; i++ {
		if tokens[i].Type != ast.TokenIdentifier || tokens[i].Value != name {
			continue
		}
		if tokens[i+1].Type == ast.TokenColonEquals {
			return &tokens[i]
		}
	}
	return nil
}

// lookupSymbolAtToken restores scope for the token index and returns LookupVariable for id.
func lookupSymbolAtToken(tc *typechecker.TypeChecker, nodes []ast.Node, allToks []ast.Token, tokIdx int, id ast.Identifier) (typechecker.Symbol, bool) {
	if err := tc.RestoreScope(nil); err != nil {
		return typechecker.Symbol{}, false
	}
	scopeNode := findInnermostScopeNode(nodes, allToks, tokIdx, tc)
	if scopeNode != nil {
		if err := tc.RestoreScope(scopeNode); err != nil {
			return typechecker.Symbol{}, false
		}
	}
	return tc.CurrentScope().LookupVariable(id)
}

// sameBinding reports whether two symbols refer to the same variable binding.
func sameBinding(a, b typechecker.Symbol) bool {
	if a.Scope == nil || b.Scope == nil {
		return false
	}
	return a.Scope == b.Scope && a.Identifier == b.Identifier
}
