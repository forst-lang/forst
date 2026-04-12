package ast

// AssignmentNode represents a variable assignment in the AST
type AssignmentNode struct {
	Node
	LValues       []ExpressionNode // Assignable expressions (VariableNode or IndexExpressionNode, etc.)
	RValues       []ExpressionNode // Values being assigned
	ExplicitTypes []*TypeNode      // Optional explicit types for each name
	IsShort       bool             // Whether this is a short := assignment
}

// Kind returns the node kind for an assignment
func (n AssignmentNode) Kind() NodeKind {
	return NodeKindAssignment
}

// String returns a string representation of the assignment
func (n AssignmentNode) String() string {
	result := "Assignment("

	// Build comma-separated list of names and types
	for i, lv := range n.LValues {
		if i > 0 {
			result += ", "
		}
		if vn, ok := lv.(VariableNode); ok {
			result += vn.String()
			if len(n.ExplicitTypes) > i && n.ExplicitTypes[i] != nil {
				result += ": " + n.ExplicitTypes[i].String()
			}
		} else {
			result += lv.String()
		}
	}

	// Add appropriate assignment operator
	if n.IsShort {
		result += " := "
	} else {
		result += " = "
	}

	// Add comma-separated list of values
	for i, val := range n.RValues {
		if i > 0 {
			result += ", "
		}
		result += val.String()
	}

	return result + ")"
}
