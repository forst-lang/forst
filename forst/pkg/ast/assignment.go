package ast

// AssignmentNode represents a variable assignment in the AST
type AssignmentNode struct {
	Names         []string         // Names being assigned to
	Values        []ExpressionNode // Values being assigned
	ExplicitTypes []*TypeNode      // Optional explicit types for each name
	IsShort       bool             // Whether this is a short := assignment
}

// NodeType returns the type of this AST node
func (n AssignmentNode) NodeType() string {
	return "Assignment"
}

// String returns a string representation of the assignment
func (n AssignmentNode) String() string {
	var result string

	// Build comma-separated list of names and types
	for i, name := range n.Names {
		if i > 0 {
			result += ", "
		}
		result += name
		if len(n.ExplicitTypes) > i && n.ExplicitTypes[i] != nil {
			result += ": " + n.ExplicitTypes[i].String()
		}
	}

	// Add appropriate assignment operator
	if n.IsShort {
		result += " := "
	} else {
		result += " = "
	}

	// Add comma-separated list of values
	for i, val := range n.Values {
		if i > 0 {
			result += ", "
		}
		result += val.String()
	}

	return result
}
