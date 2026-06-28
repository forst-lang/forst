package ast

import "sort"

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

// ShapeFieldNamesInOrder returns declaration order when set, otherwise lexicographically sorted names.
func ShapeFieldNamesInOrder(fields map[string]ShapeFieldNode, fieldOrder []string) []string {
	if len(fieldOrder) > 0 {
		return fieldOrder
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ShapeFieldNamesForCompositeEmit orders payload fields for Go composite literals: shape
// declaration order first, then any remaining payload fields in sorted order.
func ShapeFieldNamesForCompositeEmit(payloadFields map[string]ShapeFieldNode, shape *ShapeNode) []string {
	if shape != nil && len(shape.FieldOrder) > 0 {
		out := make([]string, 0, len(payloadFields))
		seen := make(map[string]struct{}, len(payloadFields))
		for _, name := range shape.FieldOrder {
			if _, ok := payloadFields[name]; !ok {
				continue
			}
			out = append(out, name)
			seen[name] = struct{}{}
		}
		var rest []string
		for name := range payloadFields {
			if _, ok := seen[name]; ok {
				continue
			}
			rest = append(rest, name)
		}
		sort.Strings(rest)
		return append(out, rest...)
	}
	return ShapeFieldNamesInOrder(payloadFields, nil)
}
