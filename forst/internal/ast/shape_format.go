package ast

// FormatShapeMemberName formats one shape field for typedef / contract source emission.
// Method contracts omit the colon: `info(msg String)`. Data fields use `name: rhs`.
func FormatShapeMemberName(name string, field ShapeFieldNode) string {
	if field.IsMethod {
		return name + field.methodSignatureString()
	}
	return name + ": " + field.String()
}

// MethodSignatureString formats a method-only contract member (`(params): returns`).
func (n ShapeFieldNode) MethodSignatureString() string {
	return n.methodSignatureString()
}
