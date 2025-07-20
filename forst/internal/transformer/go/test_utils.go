package transformergo

import (
	"bytes"
	"forst/internal/ast"
	"forst/internal/typechecker"
	"strings"

	"github.com/sirupsen/logrus"
)

func setupTestLogger() *logrus.Logger {
	return ast.SetupTestLogger()
}

func setupTypeChecker(log *logrus.Logger) *typechecker.TypeChecker {
	return typechecker.New(log, false)
}

func setupTransformer(tc *typechecker.TypeChecker, log *logrus.Logger) *Transformer {
	return New(tc, log)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Helper function to check if a string contains a substring, ignoring whitespace
func containsIgnoreWhitespace(s, substr string) bool {
	s = removeWhitespace(s)
	substr = removeWhitespace(substr)
	return contains(s, substr)
}

// Helper function to remove all whitespace from a string
func removeWhitespace(s string) string {
	var b bytes.Buffer
	for i := 0; i < len(s); i++ {
		if s[i] != ' ' && s[i] != '\n' && s[i] != '\t' && s[i] != '\r' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
