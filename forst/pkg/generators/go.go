package generators

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/token"
)

// GenerateGoCode generates Go code from a Go AST
func GenerateGoCode(goFile *goast.File) (string, error) {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	if err := format.Node(&buf, fset, goFile); err != nil {
		return "", fmt.Errorf("failed to format Go code: %w", err)
	}
	return buf.String(), nil
}

// formatGoCode formats a Go AST into formatted source code
func formatGoCode(file *goast.File) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	format.Node(&buf, fset, file)
	return buf.String()
}
