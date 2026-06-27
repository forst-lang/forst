package gateway

import (
	"bytes"
	"encoding/json"
)

// ValidateSuccessResultIfGateway runs §12 JSON validation when the discovered function is a gateway
// handler and execution succeeded with a non-empty result. Otherwise returns nil (no-op).
func ValidateSuccessResultIfGateway(isGateway, execSuccess bool, raw json.RawMessage) error {
	if !execSuccess || !isGateway {
		return nil
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return ValidateGatewayResultJSON(raw)
}
