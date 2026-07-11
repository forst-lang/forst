package lsp

import (
	"forst/internal/ast"
	"forst/internal/forstcheck"
	"forst/internal/typechecker"

	"github.com/sirupsen/logrus"
)

// typecheckForLSP runs module-level checking when the file lives in a multi-package Forst
// module so cross-package imports resolve via Forst siblings (not emitted Go stubs).
func typecheckForLSP(log *logrus.Logger, filePath string, nodes []ast.Node) (*typechecker.TypeChecker, error) {
	tc, _, err := forstcheck.TypecheckFile(log, forstcheck.TypecheckFileOpts{
		FilePath: filePath,
		Nodes:    nodes,
	})
	return tc, err
}
