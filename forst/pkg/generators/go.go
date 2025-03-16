package generators

import (
	"bytes"
	goast "go/ast"
	"go/format"
	"go/token"
)

// GenerateGoCode generates Go code from a Go AST
func GenerateGoCode(goFile *goast.File) string {
	return formatGoCode(goFile)
}

// formatGoCode formats a Go AST into formatted source code
func formatGoCode(file *goast.File) string {
	var buf bytes.Buffer
	fset := token.NewFileSet()
	format.Node(&buf, fset, file)
	return buf.String()
}