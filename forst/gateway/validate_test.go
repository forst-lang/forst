package gateway

import (
	"encoding/json"
	"testing"
)

func TestValidateGatewayResultJSON_answer(t *testing.T) {
	raw := json.RawMessage(`{"kind":"answer","status":200,"headers":{},"body":"ok"}`)
	if err := ValidateGatewayResultJSON(raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGatewayResultJSON_pass(t *testing.T) {
	raw := json.RawMessage(`{"kind":"pass","locals":{"a":1}}`)
	if err := ValidateGatewayResultJSON(raw); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGatewayResultJSON_badKind(t *testing.T) {
	raw := json.RawMessage(`{"kind":"nope"}`)
	if err := ValidateGatewayResultJSON(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateGatewayResultJSON_empty(t *testing.T) {
	if err := ValidateGatewayResultJSON(nil); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateGatewayResultJSON(json.RawMessage(`   `)); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateGatewayResultJSON_invalidJSON(t *testing.T) {
	raw := json.RawMessage(`{`)
	if err := ValidateGatewayResultJSON(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateGatewayResultJSON_answer_statusOutOfRange(t *testing.T) {
	raw := json.RawMessage(`{"kind":"answer","status":999}`)
	if err := ValidateGatewayResultJSON(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateGatewayResultJSON_answer_statusZeroRejected(t *testing.T) {
	raw := json.RawMessage(`{"kind":"answer","status":0}`)
	if err := ValidateGatewayResultJSON(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateGatewayResultJSON_pass_invalidLocalsJSON(t *testing.T) {
	raw := json.RawMessage(`{"kind":"pass","locals":[}`)
	if err := ValidateGatewayResultJSON(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateGatewayResultJSON_pass_invalidRequestJSON(t *testing.T) {
	raw := json.RawMessage(`{"kind":"pass","request":{`)
	if err := ValidateGatewayResultJSON(raw); err == nil {
		t.Fatal("expected error")
	}
}
