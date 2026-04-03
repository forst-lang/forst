package lsp

import (
	"encoding/json"
	"strings"

	"forst/internal/ast"

	"github.com/sirupsen/logrus"
)

// LSP SymbolKind subset (see LSP spec).
const (
	lspSymbolKindFunction = 12
	lspSymbolKindStruct   = 23
	lspSymbolKindMethod   = 6
)

// LspSymbolInformation is the wire shape for documentSymbol (flat) and workspace/symbol.
type LspSymbolInformation struct {
	Name          string      `json:"name"`
	Kind          int         `json:"kind"`
	Location      LSPLocation `json:"location"`
	ContainerName string      `json:"containerName,omitempty"`
}

// handleDocumentSymbol implements textDocument/documentSymbol for top-level declarations.
func (s *LSPServer) handleDocumentSymbol(request LSPRequest) LSPServerResponse {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &LSPError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	uri := params.TextDocument.URI
	ctx, ok := s.analyzeForstDocument(uri)
	if !ok || ctx == nil || ctx.ParseErr != nil || ctx.Nodes == nil {
		s.log.WithFields(logrus.Fields{
			"function": "handleDocumentSymbol",
			"uri":      uri,
		}).Debug("documentSymbol: no parsed document")
		return LSPServerResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  []LspSymbolInformation{},
		}
	}

	syms := symbolsFromParsedDocument(uri, ctx.Tokens, ctx.Nodes)
	return LSPServerResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  syms,
	}
}

func symbolsFromParsedDocument(uri string, tokens []ast.Token, nodes []ast.Node) []LspSymbolInformation {
	var out []LspSymbolInformation
	for _, n := range nodes {
		switch v := n.(type) {
		case ast.FunctionNode:
			name := string(v.Ident.ID)
			if t := findFuncNameToken(tokens, name); t != nil {
				out = append(out, LspSymbolInformation{
					Name:     name,
					Kind:     lspSymbolKindFunction,
					Location: lspLocationFromToken(uri, t),
				})
			}
		case ast.TypeDefNode:
			name := string(v.Ident)
			if strings.HasPrefix(name, "T_") {
				continue
			}
			if t := findTypeNameToken(tokens, name); t != nil {
				out = append(out, LspSymbolInformation{
					Name:     name,
					Kind:     lspSymbolKindStruct,
					Location: lspLocationFromToken(uri, t),
				})
			}
		case *ast.TypeGuardNode:
			name := string(v.Ident)
			if t := findTypeGuardNameToken(tokens, name); t != nil {
				out = append(out, LspSymbolInformation{
					Name:     name,
					Kind:     lspSymbolKindMethod,
					Location: lspLocationFromToken(uri, t),
				})
			}
		}
	}
	return out
}
