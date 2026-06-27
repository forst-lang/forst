package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ValidateGatewayResultJSON checks non-streaming §12 invoke result bytes for gateway handlers.
func ValidateGatewayResultJSON(raw json.RawMessage) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("empty result")
	}
	var probe struct {
		Kind Kind `json:"kind"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return fmt.Errorf("result JSON: %w", err)
	}
	switch probe.Kind {
	case KindAnswer:
		var g GatewayResponse
		if err := json.Unmarshal(raw, &g); err != nil {
			return err
		}
		if g.Status < 100 || g.Status > 599 {
			return fmt.Errorf("answer: status must be between 100 and 599, got %d", g.Status)
		}
		return nil
	case KindPass:
		var p struct {
			Kind    Kind            `json:"kind"`
			Locals  json.RawMessage `json:"locals"`
			Request json.RawMessage `json:"request"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		if len(p.Locals) > 0 && !json.Valid(p.Locals) {
			return fmt.Errorf("pass: locals is not valid JSON")
		}
		if len(p.Request) > 0 && !json.Valid(p.Request) {
			return fmt.Errorf("pass: request is not valid JSON")
		}
		return nil
	default:
		return fmt.Errorf("unknown result kind %q", probe.Kind)
	}
}
