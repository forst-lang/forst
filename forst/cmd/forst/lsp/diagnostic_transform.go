package lsp

import (
	"encoding/json"
	"fmt"

	transformer_go "forst/internal/transformer/go"
	"forst/internal/lexer"
	"forst/internal/parser"
)

// transformDiagnostics runs code generation and returns LSP diagnostics from transform/debug output.
func (s *LSPServer) transformDiagnostics(ctx *forstDocumentContext, filePath string) []LSPDiagnostic {
	if ctx == nil || ctx.TC == nil {
		return nil
	}
	astNodes := ctx.Nodes
	tc := ctx.TC

	transformer := transformer_go.New(tc, s.log, false)
	_, err := transformer.TransformForstFileToGo(astNodes)
	if err != nil {
		diagnostic := diagnosticForTypecheckOrTransform(fileURIForLocalPath(filePath), ctx.Content, err, "forst-transformer", ErrorCodeTransformationFailed)
		diagnostic.Message = fmt.Sprintf("Transformation error: %v", err)

		transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)
		transformerDebugger.LogError(EventTransformerError, "Code transformation failed", &ErrorInfo{
			Code:     ErrorCodeTransformationFailed,
			Message:  err.Error(),
			Severity: SeverityError,
			Suggestions: []string{
				"Check for unsupported language constructs",
				"Verify type definitions are complete",
				"Ensure all referenced types are defined",
				"Check for recursive type definitions",
			},
		})

		return []LSPDiagnostic{diagnostic}
	}

	transformerDebugger := s.debugger.GetDebugger(PhaseTransformer, filePath)
	transformerDebugger.LogEvent(EventTransformerComplete, "Code transformation completed", map[string]any{
		"file":               filePath,
		"transformer_status": "completed",
	})

	if transformerOutput, err := transformerDebugger.GetOutput(); err == nil {
		var transformerEvents []DebugEvent
		if json.Unmarshal(transformerOutput, &transformerEvents) == nil {
			s.debugEvents = append(s.debugEvents, transformerEvents...)
		}
	}

	_ = s.lspDebugger.ProcessDebugEvents()
	return s.lspDebugger.GetDiagnostics()
}

// analyzeForstContent lexes, parses, and typechecks in-memory content for compile/diagnostic paths.
func (s *LSPServer) analyzeForstContent(filePath, content string) (*forstDocumentContext, bool) {
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

	ctx := &forstDocumentContext{
		URI:      fileURIForLocalPath(filePath),
		FilePath: filePath,
		Content:  content,
		FileID:   fid,
		Tokens:   tokens,
		ParseErr: parseErr,
	}
	if parseErr != nil {
		return ctx, true
	}

	tc, checkErr := typecheckForLSP(s.log, filePath, nodes)
	ctx.Nodes = nodes
	ctx.TC = tc
	ctx.CheckErr = checkErr
	return ctx, true
}
