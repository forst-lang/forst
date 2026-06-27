package gateway

import "testing"

func TestIsGatewayHandlerSignature(t *testing.T) {
	if !IsGatewayHandlerSignature(1, "gateway.GatewayRequest", "gateway.GatewayResponse", true) {
		t.Fatal("expected gateway handler")
	}
	if IsGatewayHandlerSignature(2, "gateway.GatewayRequest", "gateway.GatewayResponse", true) {
		t.Fatal("expected false for two params")
	}
	if IsGatewayHandlerSignature(1, "gateway.GatewayRequest", "gateway.GatewayResponse", false) {
		t.Fatal("expected false when not Result")
	}
	if IsGatewayHandlerSignature(1, "int", "gateway.GatewayResponse", true) {
		t.Fatal("expected false for wrong param type")
	}
}

func TestTempModuleNeedsGatewayImport(t *testing.T) {
	if !TempModuleNeedsGatewayImport([]string{"gateway.GatewayRequest"}) {
		t.Fatal("expected true")
	}
	if TempModuleNeedsGatewayImport([]string{"string"}) {
		t.Fatal("expected false")
	}
}

func TestTypeScriptEmitType(t *testing.T) {
	if ts, ok := TypeScriptEmitType("gateway.GatewayRequest"); !ok || ts == "" {
		t.Fatalf("request: ok=%v ts=%q", ok, ts)
	}
	if _, ok := TypeScriptEmitType("other.Type"); ok {
		t.Fatal("expected false")
	}
}
