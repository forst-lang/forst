package typechecker

import (
	"fmt"
	"forst/internal/ast"
	"strings"

	logrus "github.com/sirupsen/logrus"
)

type compatKey struct {
	actual   string
	expected string
}

func compatKeyFor(actual, expected ast.TypeNode) compatKey {
	return compatKey{
		actual:   typeNodeCompatKey(actual),
		expected: typeNodeCompatKey(expected),
	}
}

func typeNodeCompatKey(t ast.TypeNode) string {
	var b strings.Builder
	b.WriteString(string(t.Ident))
	b.WriteByte(':')
	b.WriteString(fmt.Sprint(t.TypeKind))
	if t.Assertion != nil && t.Assertion.BaseType != nil {
		b.WriteByte('@')
		b.WriteString(string(*t.Assertion.BaseType))
	}
	for _, p := range t.TypeParams {
		b.WriteByte('[')
		b.WriteString(typeNodeCompatKey(p))
		b.WriteByte(']')
	}
	return b.String()
}

// IsTypeCompatible checks if a type is compatible with an expected type,
// taking into account subtypes and type guards.
func (tc *TypeChecker) IsTypeCompatible(actual ast.TypeNode, expected ast.TypeNode) bool {
	if a, ok := tc.expandTypeDefBinaryIfNeeded(actual); ok {
		actual = a
	}
	if e, ok := tc.expandTypeDefBinaryIfNeeded(expected); ok {
		expected = e
	}
	key := compatKeyFor(actual, expected)
	if tc.compatMemo != nil {
		if v, ok := tc.compatMemo[key]; ok {
			return v
		}
	}
	if tc.log.IsLevelEnabled(logrus.DebugLevel) {
		tc.log.WithFields(logrus.Fields{
			"actual":   actual.Ident,
			"expected": expected.Ident,
			"function": "IsTypeCompatible",
		}).Debug("Checking type compatibility")
	}
	result := tc.isTypeCompatibleImpl(actual, expected)
	if tc.compatMemo == nil {
		tc.compatMemo = make(map[compatKey]bool)
	}
	tc.compatMemo[key] = result
	return result
}

func (tc *TypeChecker) debugCompat(msg string, fields logrus.Fields) {
	if !tc.log.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["function"] = "IsTypeCompatible"
	tc.log.WithFields(fields).Debug(msg)
}
