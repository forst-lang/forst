package lsp

import (
	"os"
	"path/filepath"
	"strings"

	"forst/internal/ast"
	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// forstDocumentContext holds lex/parse/typecheck results for one .ft document URI.
type forstDocumentContext struct {
	URI      string
	FilePath string
	Content  string
	FileID   string
	Tokens   []ast.Token
	Nodes    []ast.Node
	ParseErr error
	TC       *typechecker.TypeChecker
	CheckErr error
	// PackageMerge is set when TC analyzed a merged AST for multiple open files in the same package.
	PackageMerge *packageMergeInfo
}

// analyzeForstDocument loads buffer text, lexes, parses, and typechecks.
// Returns ok false if the URI is not a loadable .ft file or the compiler debugger is unavailable.
// On parse failure, ParseErr is set and TC is nil; Tokens is still populated.
func (s *LSPServer) analyzeForstDocument(uri string) (ctx *forstDocumentContext, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"function": "analyzeForstDocument",
				"panic":    r,
			}).Debug("analyzeForstDocument panic recovered")
			ctx = nil
			ok = false
		}
	}()

	filePath := filePathFromDocumentURI(uri)
	if !strings.HasSuffix(filePath, ".ft") {
		return nil, false
	}

	if _, mctx, merged := s.analyzePackageGroupMerged(uri); merged && mctx != nil {
		return mctx, true
	}

	content := s.openDocumentContent(uri)
	if content == "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return nil, false
		}
		content = string(b)
	}

	cd, dbgOK := s.debugger.(*CompilerDebugger)
	if !dbgOK {
		return nil, false
	}
	packageStore := cd.packageStore
	fileID := packageStore.RegisterFile(filePath, extractPackagePath(filePath))
	fid := string(fileID)
	lex := lexer.New([]byte(content), fid, s.log)
	tokens := lex.Lex()

	psr := parser.New(tokens, fid, s.log)
	nodes, parseErr := psr.ParseFile()

	ctx = &forstDocumentContext{
		URI:      uri,
		FilePath: filePath,
		Content:  content,
		FileID:   fid,
		Tokens:   tokens,
		ParseErr: parseErr,
	}
	if parseErr != nil {
		return ctx, true
	}

	tc := typechecker.New(s.log, false)
	tc.GoWorkspaceDir = goload.FindModuleRoot(filepath.Dir(filePath))
	checkErr := tc.CheckTypes(nodes)
	ctx.Nodes = nodes
	ctx.TC = tc
	ctx.CheckErr = checkErr
	return ctx, true
}

// peerDocumentContextForCompletion returns a cached *forstDocumentContext for another open buffer when
// buffer text matches. Used only from crossBufferTopLevelCompletionItems (read-only symbol iteration);
// the current document always uses analyzeForstDocument directly so TypeChecker scope is not reused after RestoreScope.
func (s *LSPServer) peerDocumentContextForCompletion(uri string) (*forstDocumentContext, bool) {
	content := s.openDocumentContent(uri)
	if content == "" {
		return nil, false
	}
	canon := canonicalFileURI(uri)
	s.peerAnalysisMu.Lock()
	defer s.peerAnalysisMu.Unlock()
	if e, ok := s.peerAnalysisCache[canon]; ok && e.content == content && e.ctx != nil {
		return e.ctx, true
	}
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.TC == nil {
		return nil, false
	}
	s.peerAnalysisCache[canon] = peerAnalysisCacheEntry{content: content, ctx: ctx}
	return ctx, true
}

// invalidatePeerAnalysisCache drops cached peer analysis when a buffer is closed.
func (s *LSPServer) invalidatePeerAnalysisCache(uri string) {
	s.peerAnalysisMu.Lock()
	delete(s.peerAnalysisCache, uri)
	delete(s.peerAnalysisCache, canonicalFileURI(uri))
	s.peerAnalysisMu.Unlock()
}
