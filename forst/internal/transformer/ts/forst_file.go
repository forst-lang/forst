package transformerts

import (
	"fmt"
	"os"
	"path/filepath"

	"forst/internal/goload"
	"forst/internal/lexer"
	"forst/internal/parser"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// TransformForstFileOptions configures lex/parse/typecheck/transform for one .ft file.
type TransformForstFileOptions struct {
	// RelaxedTypecheck logs and continues when type checking fails (dev server / discovery).
	// When false, a typecheck error aborts the pipeline (CLI generate).
	RelaxedTypecheck bool
}

// TransformForstFileFromPath reads path, runs the compiler pipeline, and returns TypeScript output
// for that file (types, function signatures, and per-file client code).
func TransformForstFileFromPath(filePath string, log *logrus.Logger, opts TransformForstFileOptions) (*TypeScriptOutput, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	l := lexer.New(content, filePath, log)
	tokens := l.Lex()

	p := parser.New(tokens, filePath, log)
	nodes, err := p.ParseFile()
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	tc := typechecker.New(log, false)
	tc.GoWorkspaceDir = goload.GoWorkspaceForPackages(filePath)
	if err := tc.CheckTypes(nodes); err != nil {
		if !opts.RelaxedTypecheck {
			return nil, fmt.Errorf("failed to type check: %w", err)
		}
		if log != nil {
			log.Debugf("Type checking failed for %s: %v", filePath, err)
		}
	}

	base := filepath.Base(filePath)
	stem := base[:len(base)-len(filepath.Ext(base))]

	transformer := New(tc, log)
	return transformer.TransformForstFileToTypeScript(nodes, stem)
}
