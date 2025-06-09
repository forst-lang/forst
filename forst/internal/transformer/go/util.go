package transformergo

// Helper to check if a type name is a Go built-in type
func isGoBuiltinType(typeName string) bool {
	switch typeName {
	case "string", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64", "complex64", "complex128", "bool", "byte", "rune", "error", "void":
		return true
	default:
		return false
	}
}
