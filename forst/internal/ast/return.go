package ast

import "fmt"

// ReturnNode represents a return statement
type ReturnNode struct {
	Values []ExpressionNode
	Type   TypeNode
}

// Kind returns the node kind for a return statement
func (r ReturnNode) Kind() NodeKind {
	return NodeKindReturn
}

// String returns a string representation of the return statement
func (r ReturnNode) String() string {
	if len(r.Values) == 0 {
		return "Return()"
	}
	if len(r.Values) == 1 {
		return fmt.Sprintf("Return(%s)", r.Values[0].String())
	}

	values := make([]string, len(r.Values))
	for i, val := range r.Values {
		values[i] = val.String()
	}
	return fmt.Sprintf("Return(%s)", joinStrings(values, ", "))
}

// Helper function to join strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
