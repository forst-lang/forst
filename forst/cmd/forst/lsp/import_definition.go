package lsp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
)

// qualifiedImportSymbolAt reports pkgLocal.symbol when tokIdx is the right-hand identifier in an import qualifier.
func qualifiedImportSymbolAt(tokens []ast.Token, tokIdx int) (pkgLocal, symbol string, ok bool) {
	if tokIdx < 2 || tokens[tokIdx].Type != ast.TokenIdentifier {
		return "", "", false
	}
	if tokens[tokIdx-1].Type != ast.TokenDot || tokens[tokIdx-2].Type != ast.TokenIdentifier {
		return "", "", false
	}
	return tokens[tokIdx-2].Value, tokens[tokIdx].Value, true
}

func importPathForPackageDir(moduleRoot, modulePath, pkgDir string) string {
	moduleRoot = canonicalLocalPath(moduleRoot)
	pkgDir = canonicalLocalPath(pkgDir)
	if moduleRoot == "" || modulePath == "" || pkgDir == "" {
		return ""
	}
	rel, err := filepath.Rel(moduleRoot, pkgDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return modulePath
	}
	return modulePath + "/" + rel
}

func pathWithinModuleRoot(path, moduleRoot string) bool {
	path = canonicalLocalPath(path)
	moduleRoot = canonicalLocalPath(moduleRoot)
	if path == moduleRoot {
		return true
	}
	return strings.HasPrefix(path, moduleRoot+string(os.PathSeparator))
}

func (s *LSPServer) forstFileURIsUnderModule(moduleRoot string) []string {
	moduleRoot = canonicalLocalPath(moduleRoot)
	if moduleRoot == "" {
		return nil
	}
	var out []string
	seen := make(map[string]struct{})
	s.documentMu.RLock()
	for u := range s.openDocuments {
		p := filePathFromDocumentURI(u)
		if p == "" || !strings.HasSuffix(p, ".ft") || !pathWithinModuleRoot(p, moduleRoot) {
			continue
		}
		cu := canonicalFileURI(u)
		if _, ok := seen[cu]; ok {
			continue
		}
		out = append(out, cu)
		seen[cu] = struct{}{}
	}
	s.documentMu.RUnlock()

	_ = filepath.WalkDir(moduleRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".ft") {
			return nil
		}
		u := fileURIForLocalPath(path)
		if _, ok := seen[u]; ok {
			return nil
		}
		out = append(out, u)
		seen[u] = struct{}{}
		return nil
	})
	sort.Strings(out)
	return out
}

// collectCrossPackageReferencesForExportedFunction finds qualified import call sites
// (e.g. alpha.LogExpiry) that resolve to the callee package directory of ctx.
func (s *LSPServer) collectCrossPackageReferencesForExportedFunction(ctx *forstDocumentContext, calleeName string) []LSPLocation {
	if ctx == nil || ctx.TC == nil || calleeName == "" {
		return nil
	}
	moduleRoot := ctx.TC.GoWorkspaceDir
	if moduleRoot == "" {
		moduleRoot = goload.FindModuleRoot(filepath.Dir(ctx.FilePath))
	}
	if moduleRoot == "" {
		return nil
	}
	modulePath := goload.ModulePath(moduleRoot)
	calleeDir := canonicalDirForPath(ctx.FilePath)
	calleeImportPath := importPathForPackageDir(moduleRoot, modulePath, calleeDir)
	if calleeImportPath == "" {
		return nil
	}

	var out []LSPLocation
	for _, u := range s.forstFileURIsUnderModule(moduleRoot) {
		p := filePathFromDocumentURI(u)
		if p == "" || canonicalDirForPath(p) == calleeDir {
			continue
		}
		peerCtx, ok := s.analyzeForstDocument(u)
		if !ok || peerCtx == nil || peerCtx.ParseErr != nil || peerCtx.TC == nil {
			continue
		}
		for i := range peerCtx.Tokens {
			t := &peerCtx.Tokens[i]
			if t.Type != ast.TokenIdentifier || t.Value != calleeName {
				continue
			}
			pkgLocal, sym, ok := qualifiedImportSymbolAt(peerCtx.Tokens, i)
			if !ok || sym != calleeName {
				continue
			}
			if goImportLocalShadowedByForstVar(peerCtx.TC, pkgLocal) || !peerCtx.TC.IsImportedLocalName(pkgLocal) {
				continue
			}
			importPath, ok := peerCtx.TC.ImportPathForLocal(pkgLocal)
			if !ok || importPath != calleeImportPath {
				continue
			}
			out = append(out, lspLocationFromToken(u, t))
		}
	}
	return out
}

func dirForModuleImportPath(moduleRoot, modulePath, importPath string) string {
	if moduleRoot == "" || modulePath == "" || importPath == "" {
		return ""
	}
	if importPath == modulePath {
		return moduleRoot
	}
	prefix := modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return ""
	}
	rel := strings.TrimPrefix(importPath, prefix)
	return filepath.Join(moduleRoot, filepath.FromSlash(rel))
}

func (s *LSPServer) definingLocationForQualifiedImport(ctx *forstDocumentContext, tokIdx int) *LSPLocation {
	pkgLocal, symbol, ok := qualifiedImportSymbolAt(ctx.Tokens, tokIdx)
	if !ok {
		return nil
	}
	tc := ctx.TC
	if tc == nil {
		return nil
	}
	if goImportLocalShadowedByForstVar(tc, pkgLocal) || !tc.IsImportedLocalName(pkgLocal) {
		return nil
	}
	importPath, ok := tc.ImportPathForLocal(pkgLocal)
	if !ok {
		return nil
	}
	moduleRoot := tc.GoWorkspaceDir
	if moduleRoot == "" {
		moduleRoot = goload.FindModuleRoot(filepath.Dir(ctx.FilePath))
	}
	if moduleRoot == "" {
		return nil
	}
	targetDir := dirForModuleImportPath(moduleRoot, goload.ModulePath(moduleRoot), importPath)
	if targetDir == "" {
		return nil
	}
	return s.findTopLevelSymbolInForstDir(targetDir, symbol)
}

func (s *LSPServer) findTopLevelSymbolInForstDir(dir, symbol string) *LSPLocation {
	for _, u := range s.forstFileURIsInDir(dir) {
		tokens := s.tokensForURI(u)
		if tokens == nil {
			continue
		}
		if defTok := findFuncNameToken(tokens, symbol); defTok != nil {
			return lspLocationPtrFromToken(u, defTok)
		}
		if defTok := findTypeNameToken(tokens, symbol); defTok != nil {
			return lspLocationPtrFromToken(u, defTok)
		}
		if defTok := findTypeGuardNameToken(tokens, symbol); defTok != nil {
			return lspLocationPtrFromToken(u, defTok)
		}
	}
	return nil
}

func (s *LSPServer) forstFileURIsInDir(dir string) []string {
	dir = canonicalLocalPath(dir)
	var out []string
	seen := make(map[string]struct{})
	s.documentMu.RLock()
	for u := range s.openDocuments {
		p := filePathFromDocumentURI(u)
		if p == "" || !strings.HasSuffix(p, ".ft") {
			continue
		}
		if canonicalDirForPath(p) != dir {
			continue
		}
		cu := canonicalFileURI(u)
		if _, ok := seen[cu]; ok {
			continue
		}
		out = append(out, cu)
		seen[cu] = struct{}{}
	}
	s.documentMu.RUnlock()

	entries, err := os.ReadDir(dir)
	if err != nil {
		sort.Strings(out)
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ft") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		u := fileURIForLocalPath(full)
		if _, ok := seen[u]; ok {
			continue
		}
		out = append(out, u)
		seen[u] = struct{}{}
	}
	sort.Strings(out)
	return out
}

func (s *LSPServer) tokensForURI(uri string) []ast.Token {
	content := s.openDocumentContent(uri)
	if content == "" {
		p := filePathFromDocumentURI(uri)
		if p == "" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		content = string(b)
	}
	cd, ok := s.debugger.(*CompilerDebugger)
	if !ok {
		return nil
	}
	fp := filePathFromDocumentURI(uri)
	if fp == "" {
		return nil
	}
	fid := string(cd.packageStore.RegisterFile(fp, extractPackagePath(fp)))
	return lexer.New([]byte(content), fid, s.log).Lex()
}
