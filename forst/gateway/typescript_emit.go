package gateway

// TypeScriptEmitType returns the emitted TypeScript type for qualified forst/gateway names used in
// forst generate (maps to @forst/sidecar §12 wire types). The second return is false if ident is not
// a known gateway stdlib qualified name.
func TypeScriptEmitType(goQualifiedIdent string) (string, bool) {
	switch goQualifiedIdent {
	case "gateway.GatewayRequest":
		return "import('@forst/sidecar').ForstRoutedRequest", true
	case "gateway.GatewayResponse":
		return "import('@forst/sidecar').ForstRoutedResponse", true
	default:
		return "", false
	}
}
