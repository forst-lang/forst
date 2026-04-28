package gateway

import "strings"

// IsGatewayHandlerSignature reports whether resolved types match a GatewayHandler-style function:
// exactly one parameter typed like GatewayRequest and a Result whose success type references GatewayResponse.
// Param and result strings are the same display forms discovery uses (e.g. gateway.GatewayRequest).
func IsGatewayHandlerSignature(paramCount int, paramResolvedType, resultSuccessType string, isResultReturn bool) bool {
	if paramCount != 1 || !isResultReturn {
		return false
	}
	pt := paramResolvedType
	st := resultSuccessType
	return strings.Contains(pt, "GatewayRequest") && strings.Contains(st, "GatewayResponse")
}
