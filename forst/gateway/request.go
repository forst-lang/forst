// Package gateway defines Forst HTTP gateway request/response types and §12 JSON encoding.
package gateway

// GatewayRequest is the invoke argument snapshot aligned with @forst/sidecar ForstRoutedRequest.
type GatewayRequest struct {
	Method     string         `json:"method"`
	URL        string         `json:"url"`
	Path       string         `json:"path"`
	Query      map[string]any `json:"query"`
	Headers    map[string]any `json:"headers"`
	BodyBase64 string         `json:"bodyBase64"`
}
