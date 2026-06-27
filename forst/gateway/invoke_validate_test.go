package gateway

import (
	"encoding/json"
	"testing"
)

func TestValidateSuccessResultIfGateway_skipsWhenNotGateway(t *testing.T) {
	err := ValidateSuccessResultIfGateway(false, true, json.RawMessage(`{"kind":"nope"}`))
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateSuccessResultIfGateway_validatesWhenGateway(t *testing.T) {
	err := ValidateSuccessResultIfGateway(true, true, json.RawMessage(`{"kind":"nope"}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
