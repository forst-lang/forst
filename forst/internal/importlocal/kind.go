package importlocal

// Kind distinguishes Go package imports from Node/TypeScript imports.
type Kind int

const (
	KindGo Kind = iota
	KindNode
)

func (k Kind) diagnosticLabel() string {
	switch k {
	case KindGo:
		return "Go"
	case KindNode:
		return "node"
	default:
		return "import"
	}
}

func (k Kind) isReserved(name string) bool {
	switch k {
	case KindGo:
		return IsReservedGoImportLocal(name)
	case KindNode:
		return IsReservedNodeImportLocal(name)
	default:
		return IsReservedNodeImportLocal(name)
	}
}
