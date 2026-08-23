package pb

import (
	"fmt"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestWireMessages_marshalUnmarshalRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("Frame_request_body", func(t *testing.T) {
		t.Parallel()
		in := &Frame{
			Id: 42,
			Body: &Frame_Request{
				Request: &WireRequest{
					Method:      "initialize",
					PayloadJson: []byte(`{"v":1}`),
				},
			},
		}
		roundtripFrame(t, in)
	})

	t.Run("Frame_response_body", func(t *testing.T) {
		t.Parallel()
		in := &Frame{
			Id: 7,
			Body: &Frame_Response{
				Response: &WireResponse{
					Result: &WireResponse_OkJson{OkJson: []byte(`{"ok":true}`)},
				},
			},
		}
		roundtripFrame(t, in)
	})

	t.Run("WireRequest", func(t *testing.T) {
		t.Parallel()
		in := &WireRequest{Method: "ping", PayloadJson: []byte("null")}
		out := roundtrip(t, in, &WireRequest{})
		if out.GetMethod() != in.Method || string(out.GetPayloadJson()) != string(in.PayloadJson) {
			t.Fatalf("roundtrip mismatch: %+v vs %+v", in, out)
		}
	})

	t.Run("WireResponse_ok_json", func(t *testing.T) {
		t.Parallel()
		in := &WireResponse{Result: &WireResponse_OkJson{OkJson: []byte(`[1,2]`)}}
		out := roundtrip(t, in, &WireResponse{})
		if string(out.GetOkJson()) != string(in.GetOkJson()) || out.GetErr() != nil {
			t.Fatalf("ok roundtrip mismatch: %+v", out)
		}
	})

	t.Run("WireResponse_err", func(t *testing.T) {
		t.Parallel()
		in := &WireResponse{
			Result: &WireResponse_Err{
				Err: &ErrorDetail{Code: 9, Message: "boom", DataJson: []byte(`{"x":1}`)},
			},
		}
		out := roundtrip(t, in, &WireResponse{})
		errDetail := out.GetErr()
		if errDetail == nil || errDetail.Code != 9 || errDetail.Message != "boom" {
			t.Fatalf("err roundtrip mismatch: %+v", out)
		}
	})

	t.Run("ErrorDetail", func(t *testing.T) {
		t.Parallel()
		in := &ErrorDetail{Code: 1, Message: "m", DataJson: []byte("{}")}
		out := roundtrip(t, in, &ErrorDetail{})
		if out.GetCode() != 1 || out.GetMessage() != "m" || string(out.GetDataJson()) != "{}" {
			t.Fatalf("ErrorDetail roundtrip mismatch: %+v", out)
		}
	})

	t.Run("InitializeResult", func(t *testing.T) {
		t.Parallel()
		in := &InitializeResult{Protocol: "forst-node-v2"}
		out := roundtrip(t, in, &InitializeResult{})
		if out.GetProtocol() != in.Protocol {
			t.Fatalf("InitializeResult roundtrip mismatch: %+v", out)
		}
	})

	t.Run("GenNextBatchResult", func(t *testing.T) {
		t.Parallel()
		in := &GenNextBatchResult{
			Steps: []*GenStepWire{
				{Kind: "log", Message: "hi", ValueJson: []byte("1"), DataJson: []byte("2")},
			},
		}
		out := roundtrip(t, in, &GenNextBatchResult{})
		steps := out.GetSteps()
		if len(steps) != 1 || steps[0].GetKind() != "log" || steps[0].GetMessage() != "hi" {
			t.Fatalf("GenNextBatchResult roundtrip mismatch: %+v", out)
		}
	})

	t.Run("GenStepWire", func(t *testing.T) {
		t.Parallel()
		in := &GenStepWire{Kind: "step", ValueJson: []byte("v"), Message: "msg", DataJson: []byte("d")}
		out := roundtrip(t, in, &GenStepWire{})
		if out.GetKind() != in.Kind || out.GetMessage() != in.Message {
			t.Fatalf("GenStepWire roundtrip mismatch: %+v", out)
		}
	})
}

func TestWireMessages_protoReflectAndGetters(t *testing.T) {
	t.Parallel()

	assertProtoMethods := func(t *testing.T, msg proto.Message) {
		t.Helper()
		if fmt.Sprint(msg) == "" {
			t.Fatal("String() returned empty")
		}
		if msg.ProtoReflect().Descriptor().ParentFile().Path() == "" {
			t.Fatal("Descriptor path empty")
		}
		if d, _ := msg.(interface{ Descriptor() ([]byte, []int) }).Descriptor(); len(d) == 0 {
			t.Fatal("Descriptor() returned empty bytes")
		}
		msg.ProtoReflect()
	}

	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		frame := &Frame{
			Id: 1,
			Body: &Frame_Request{Request: &WireRequest{Method: "m", PayloadJson: []byte("p")}},
		}
		assertProtoMethods(t, frame)
		if frame.GetId() != 1 || frame.GetRequest().GetMethod() != "m" || frame.GetResponse() != nil {
			t.Fatalf("Frame getters: id=%d req=%v resp=%v", frame.GetId(), frame.GetRequest(), frame.GetResponse())
		}

		respFrame := &Frame{Body: &Frame_Response{Response: &WireResponse{Result: &WireResponse_Err{Err: &ErrorDetail{Code: 2}}}}}
		if respFrame.GetRequest() != nil || respFrame.GetResponse().GetErr().GetCode() != 2 {
			t.Fatalf("Frame response getters mismatch")
		}

		req := &WireRequest{Method: "x", PayloadJson: []byte("y")}
		assertProtoMethods(t, req)
		if req.GetMethod() != "x" || string(req.GetPayloadJson()) != "y" {
			t.Fatal("WireRequest getters")
		}

		okResp := &WireResponse{Result: &WireResponse_OkJson{OkJson: []byte("ok")}}
		assertProtoMethods(t, okResp)
		if string(okResp.GetOkJson()) != "ok" || okResp.GetErr() != nil {
			t.Fatal("WireResponse ok getters")
		}

		errResp := &WireResponse{Result: &WireResponse_Err{Err: &ErrorDetail{Message: "e"}}}
		if errResp.GetOkJson() != nil || errResp.GetErr().GetMessage() != "e" {
			t.Fatal("WireResponse err getters")
		}

		errDetail := &ErrorDetail{Code: 3, Message: "m", DataJson: []byte("d")}
		assertProtoMethods(t, errDetail)
		if errDetail.GetCode() != 3 || errDetail.GetMessage() != "m" || string(errDetail.GetDataJson()) != "d" {
			t.Fatal("ErrorDetail getters")
		}

		init := &InitializeResult{Protocol: "p"}
		assertProtoMethods(t, init)
		if init.GetProtocol() != "p" {
			t.Fatal("InitializeResult getters")
		}

		batch := &GenNextBatchResult{Steps: []*GenStepWire{{Kind: "k"}}}
		assertProtoMethods(t, batch)
		if len(batch.GetSteps()) != 1 {
			t.Fatal("GenNextBatchResult getters")
		}

		step := &GenStepWire{Kind: "k", ValueJson: []byte("v"), Message: "m", DataJson: []byte("d")}
		assertProtoMethods(t, step)
		if step.GetKind() != "k" || step.GetMessage() != "m" {
			t.Fatal("GenStepWire getters")
		}
	})

	t.Run("nil_receivers", func(t *testing.T) {
		t.Parallel()
		var frame *Frame
		if frame.GetId() != 0 || frame.GetBody() != nil || frame.GetRequest() != nil || frame.GetResponse() != nil {
			t.Fatal("nil Frame getters")
		}
		_ = (*Frame)(nil).ProtoReflect()

		var req *WireRequest
		if req.GetMethod() != "" || req.GetPayloadJson() != nil {
			t.Fatal("nil WireRequest getters")
		}

		var resp *WireResponse
		if resp.GetResult() != nil || resp.GetOkJson() != nil || resp.GetErr() != nil {
			t.Fatal("nil WireResponse getters")
		}

		var errDetail *ErrorDetail
		if errDetail.GetCode() != 0 || errDetail.GetMessage() != "" || errDetail.GetDataJson() != nil {
			t.Fatal("nil ErrorDetail getters")
		}

		var init *InitializeResult
		if init.GetProtocol() != "" {
			t.Fatal("nil InitializeResult getters")
		}

		var batch *GenNextBatchResult
		if batch.GetSteps() != nil {
			t.Fatal("nil GenNextBatchResult getters")
		}

		var step *GenStepWire
		if step.GetKind() != "" || step.GetValueJson() != nil || step.GetMessage() != "" || step.GetDataJson() != nil {
			t.Fatal("nil GenStepWire getters")
		}
	})

	t.Run("Reset_clears_fields", func(t *testing.T) {
		t.Parallel()
		frame := &Frame{
			Id: 99,
			Body: &Frame_Request{Request: &WireRequest{Method: "m"}},
		}
		frame.Reset()
		if frame.Id != 0 || frame.Body != nil {
			t.Fatalf("Frame.Reset: %+v", frame)
		}

		req := &WireRequest{Method: "m", PayloadJson: []byte("x")}
		req.Reset()
		if req.Method != "" || req.PayloadJson != nil {
			t.Fatalf("WireRequest.Reset: %+v", req)
		}
	})
}

func roundtripFrame(t *testing.T, in *Frame) {
	t.Helper()
	out := roundtrip(t, in, &Frame{})
	if out.Id != in.Id {
		t.Fatalf("Frame id: got %d want %d", out.Id, in.Id)
	}
	switch inBody := in.Body.(type) {
	case *Frame_Request:
		got := out.GetRequest()
		if got == nil || got.Method != inBody.Request.Method {
			t.Fatalf("Frame request roundtrip: %+v", out)
		}
	case *Frame_Response:
		got := out.GetResponse()
		if got == nil {
			t.Fatalf("Frame response roundtrip missing body: %+v", out)
		}
	default:
		t.Fatalf("unexpected frame body type %T", in.Body)
	}
}

func roundtrip[T proto.Message](t *testing.T, in T, out T) T {
	t.Helper()
	data, err := proto.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := proto.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
