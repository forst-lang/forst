package importlocal

// Kind distinguishes Go package imports from JavaScript bridge imports.
type Kind int

const (
	KindGo Kind = iota
	KindBridge
)

func (k Kind) diagnosticLabel() string {
	switch k {
	case KindGo:
		return "Go"
	case KindBridge:
		return "JS"
	default:
		return "import"
	}
}

func (k Kind) isReserved(name string) bool {
	switch k {
	case KindGo:
		return IsReservedGoImportLocal(name)
	case KindBridge:
		return IsReservedBridgeImportLocal(name)
	default:
		return IsReservedBridgeImportLocal(name)
	}
}
