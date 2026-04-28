package gateway

import "encoding/json"

// Kind discriminates §12 success result JSON for gateways.
type Kind string

const (
	KindAnswer Kind = "answer"
	KindPass   Kind = "pass"
)

// GatewayResponse is the Ok payload for Result(GatewayResponse, E); it maps 1:1 to §12 JSON.
type GatewayResponse struct {
	Kind Kind `json:"kind"`

	// answer (kind == "answer")
	Status     int               `json:"status,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	BodyBase64 string            `json:"bodyBase64,omitempty"`

	// pass (kind == "pass") — optional locals / request merge for Express.
	Locals   json.RawMessage `json:"locals,omitempty"`
	Request  json.RawMessage `json:"request,omitempty"`
}

// AnswerText returns kind "answer" with a UTF-8 body (typical 200 text/html).
func AnswerText(status int, headers map[string]string, body string) GatewayResponse {
	if status == 0 {
		status = 200
	}
	if headers == nil {
		headers = map[string]string{}
	}
	return GatewayResponse{
		Kind:    KindAnswer,
		Status:  status,
		Headers: headers,
		Body:    body,
	}
}

// AnswerBytes returns kind "answer" with a raw body transported as base64 (§12 bodyBase64).
func AnswerBytes(status int, headers map[string]string, bodyBase64 string) GatewayResponse {
	if headers == nil {
		headers = map[string]string{}
	}
	return GatewayResponse{
		Kind:       KindAnswer,
		Status:     status,
		Headers:    headers,
		BodyBase64: bodyBase64,
	}
}

// Pass delegates to Express next(); optional JSON objects are marshalled for locals/request.
func Pass(locals, request map[string]any) GatewayResponse {
	g := GatewayResponse{Kind: KindPass}
	if locals != nil {
		g.Locals, _ = json.Marshal(locals)
	}
	if request != nil {
		g.Request, _ = json.Marshal(request)
	}
	return g
}

// PassEmpty is Pass with no merges.
func PassEmpty() GatewayResponse {
	return GatewayResponse{Kind: KindPass}
}

// Plain200Text returns kind "answer" with status 200, no extra headers, and a UTF-8 body.
// Single-string helper for Forst call sites that avoid nil map literals.
func Plain200Text(body string) GatewayResponse {
	return AnswerText(200, nil, body)
}
