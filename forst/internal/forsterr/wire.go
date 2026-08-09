package forsterr

import "encoding/json"

// WireError is the structured error payload on the invoke HTTP envelope (contract v2).
type WireError struct {
	Tag     string          `json:"tag"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Message string          `json:"message,omitempty"`
}

// Tagged is implemented by generated nominal Forst error types.
type Tagged interface {
	error
	ForstErrorTag() string
}

// Encode maps a nominal Forst error to WireError when ForstErrorTag is present.
func Encode(err error) (*WireError, bool) {
	if err == nil {
		return nil, false
	}
	t, ok := err.(Tagged)
	if !ok {
		return nil, false
	}
	payload, _ := json.Marshal(t)
	return &WireError{
		Tag:     t.ForstErrorTag(),
		Payload: payload,
		Message: err.Error(),
	}, true
}
