package nodert

import (
	"bytes"
	"encoding/json"
	"testing"

	"forst/nodert/pb"
)

func TestWriteReadProtoFrame_roundTrip(t *testing.T) {
	params := CallParams{
		ModuleID:   "legacy/payment.ts",
		ExportName: "create",
		Args:       json.RawMessage(`[1,"usd"]`),
	}
	frame, err := newProtoRequestFrame(7, MethodCall, params)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteProtoFrame(&buf, frame, DefaultMaxMsgLen); err != nil {
		t.Fatal(err)
	}
	got, err := ReadProtoFrame(&buf, DefaultMaxMsgLen)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetId() != 7 {
		t.Fatalf("id = %d", got.GetId())
	}
	req := got.GetRequest()
	if req == nil || req.Method != MethodCall {
		t.Fatalf("request = %+v", req)
	}
}

func TestPickWireProtocol_requiresProtoV1(t *testing.T) {
	got := pickWireProtocol(
		[]string{WireProtocolProtoV1},
		[]string{WireProtocolProtoV1},
	)
	if got != WireProtocolProtoV1 {
		t.Fatalf("got %q", got)
	}
	if pickWireProtocol([]string{"other"}, []string{WireProtocolProtoV1}) != "" {
		t.Fatal("expected empty when client lacks proto support")
	}
}

func TestProtoOKResponse_roundTrip(t *testing.T) {
	frame, err := protoOKResponse(3, InitializeResult{OK: true, Protocol: WireProtocolProtoV1})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := decodeProtoOK(frame)
	if err != nil {
		t.Fatal(err)
	}
	var init InitializeResult
	if err := json.Unmarshal(raw, &init); err != nil {
		t.Fatal(err)
	}
	if init.Protocol != WireProtocolProtoV1 {
		t.Fatalf("protocol = %q", init.Protocol)
	}
}

func TestProtoErrorResponse_decode(t *testing.T) {
	frame := protoErrorResponse(1, -32001, "forbidden", nil)
	_, err := decodeProtoOK(frame)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFrame_pbPackage(t *testing.T) {
	var f pb.Frame
	f.Id = 1
	if f.GetId() != 1 {
		t.Fatal("pb package broken")
	}
}
