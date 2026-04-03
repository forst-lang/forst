package lsp

import (
	"os"
	"runtime"
	"strings"

	"forst/internal/ast"
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

	filePath := strings.TrimPrefix(uri, "file://")
	if runtime.GOOS == "windows" {
		filePath = strings.TrimPrefix(filePath, "/")
	}
	if !strings.HasSuffix(filePath, ".ft") {
		return nil, false
	}

	s.documentMu.RLock()
	content, haveOpen := s.openDocuments[uri]
	s.documentMu.RUnlock()
	if !haveOpen || content == "" {
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
	checkErr := tc.CheckTypes(nodes)
	ctx.Nodes = nodes
	ctx.TC = tc
	ctx.CheckErr = checkErr
	return ctx, true
}
