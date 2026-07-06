package typechecker

import (
	"forst/internal/ast"
	"forst/internal/hasher"
	"sort"
	"strings"
)

type shapeAliasIndex struct {
	byShapeHash          map[hasher.NodeHash]ast.TypeIdent
	byAssertionTypeIdent map[ast.TypeIdent]ast.TypeIdent
}

func (tc *TypeChecker) shapeAliasIndexOrBuild() *shapeAliasIndex {
	if tc.shapeAliasIndex != nil {
		return tc.shapeAliasIndex
	}
	idx := &shapeAliasIndex{
		byShapeHash:          make(map[hasher.NodeHash]ast.TypeIdent),
		byAssertionTypeIdent: make(map[ast.TypeIdent]ast.TypeIdent),
	}
	shapeCandidates := make(map[hasher.NodeHash][]ast.TypeIdent)
	assertionCandidates := make(map[ast.TypeIdent][]ast.TypeIdent)
	for _, def := range tc.Defs {
		userDef, ok := def.(ast.TypeDefNode)
		if !ok || userDef.Ident == "" || strings.HasPrefix(string(userDef.Ident), "T_") {
			continue
		}
		if payload, ok := ast.PayloadShape(userDef.Expr); ok {
			h, err := tc.Hasher.HashNode(*payload)
			if err != nil {
				continue
			}
			shapeCandidates[h] = append(shapeCandidates[h], userDef.Ident)
		}
		if _, ok := typeDefAssertionFromExpr(userDef.Expr); ok {
			bt := userDef.Ident
			a := ast.AssertionNode{BaseType: &bt}
			h, err := tc.Hasher.HashNode(a)
			if err != nil {
				continue
			}
			key := h.ToTypeIdent()
			assertionCandidates[key] = append(assertionCandidates[key], userDef.Ident)
		}
	}
	for h, idents := range shapeCandidates {
		idx.byShapeHash[h] = stableTypeIdentWinner(idents)
	}
	for key, idents := range assertionCandidates {
		idx.byAssertionTypeIdent[key] = stableTypeIdentWinner(idents)
	}
	tc.shapeAliasIndex = idx
	return idx
}

func stableTypeIdentWinner(idents []ast.TypeIdent) ast.TypeIdent {
	if len(idents) == 0 {
		return ""
	}
	sort.Slice(idents, func(i, j int) bool {
		return idents[i] < idents[j]
	})
	return idents[0]
}

func (tc *TypeChecker) lookupShapeAliasForHashType(typeNode ast.TypeNode) (ast.TypeIdent, bool) {
	hashDef, ok := tc.Defs[typeNode.Ident]
	if !ok {
		return "", false
	}
	hashTypeDef, ok := hashDef.(ast.TypeDefNode)
	if !ok {
		return "", false
	}
	hashShapeExpr, ok := hashTypeDef.Expr.(ast.TypeDefShapeExpr)
	if !ok {
		return "", false
	}
	h, err := tc.Hasher.HashNode(hashShapeExpr.Shape)
	if err != nil {
		return "", false
	}
	alias, ok := tc.shapeAliasIndexOrBuild().byShapeHash[h]
	return alias, ok
}

func (tc *TypeChecker) lookupAssertionAliasForHashIdent(ident ast.TypeIdent) (ast.TypeIdent, bool) {
	alias, ok := tc.shapeAliasIndexOrBuild().byAssertionTypeIdent[ident]
	return alias, ok
}
