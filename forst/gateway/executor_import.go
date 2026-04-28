package gateway

import "strings"

// TempModuleNeedsGatewayImport reports whether executor-generated Go should import forst/gateway
// (e.g. main wrapper references gateway.GatewayRequest).
func TempModuleNeedsGatewayImport(paramTypes []string) bool {
	for _, t := range paramTypes {
		if strings.Contains(t, "gateway.") {
			return true
		}
	}
	return false
}
