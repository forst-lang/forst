package nodert

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	"forst/nodert/pb"
	
)

// WireProtocolProtoV1 is the protobuf-framed wire protocol version this Go
// supervisor speaks; pickWireProtocol negotiates it against the versions the
// Node host advertises before falling back to plain JSON framing.
const WireProtocolProtoV1 = "forst-node-proto-v1"

// WriteProtoFrame marshals a protobuf Frame with a 4-byte big-endian length prefix.
func WriteProtoFrame(w io.Writer, frame *pb.Frame, maxLen int) error {
	if maxLen <= 0 {
		maxLen = DefaultMaxMsgLen
	}
	data, err := proto.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal proto frame: %w", err)
	}
	if len(data) > maxLen {
		return fmt.Errorf("proto frame exceeds max size %d bytes", maxLen)
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(data)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("write proto frame length: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write proto frame body: %w", err)
	}
	return nil
}

// ReadProtoFrame reads a length-prefixed protobuf Frame.
func ReadProtoFrame(r io.Reader, maxLen int) (*pb.Frame, error) {
	if maxLen <= 0 {
		maxLen = DefaultMaxMsgLen
	}
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("read proto frame length: %w", err)
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 {
		return nil, fmt.Errorf("empty proto frame")
	}
	if int(n) > maxLen {
		return nil, fmt.Errorf("proto frame exceeds max size %d bytes", maxLen)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read proto frame body: %w", err)
	}
	var frame pb.Frame
	if err := proto.Unmarshal(buf, &frame); err != nil {
		return nil, fmt.Errorf("unmarshal proto frame: %w", err)
	}
	return &frame, nil
}

func newProtoRequestFrame(id uint64, method string, params any) (*pb.Frame, error) {
	payload, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal request payload: %w", err)
	}
	return &pb.Frame{
		Id: id,
		Body: &pb.Frame_Request{
			Request: &pb.WireRequest{
				Method:      method,
				PayloadJson: payload,
			},
		},
	}, nil
}

func protoOKResponse(id uint64, result any) (*pb.Frame, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal response payload: %w", err)
	}
	return &pb.Frame{
		Id: id,
		Body: &pb.Frame_Response{
			Response: &pb.WireResponse{
				Result: &pb.WireResponse_OkJson{OkJson: payload},
			},
		},
	}, nil
}

func protoErrorResponse(id uint64, code int32, message string, data json.RawMessage) *pb.Frame {
	return &pb.Frame{
		Id: id,
		Body: &pb.Frame_Response{
			Response: &pb.WireResponse{
				Result: &pb.WireResponse_Err{
					Err: &pb.ErrorDetail{
						Code:      code,
						Message:   message,
						DataJson:  data,
					},
				},
			},
		},
	}
}

func decodeProtoOK(frame *pb.Frame) (json.RawMessage, error) {
	resp := frame.GetResponse()
	if resp == nil {
		return nil, fmt.Errorf("proto frame missing response")
	}
	if errDetail := resp.GetErr(); errDetail != nil {
		return nil, rpcErrorFromWire(int(errDetail.Code), errDetail.Message, errDetail.DataJson)
	}
	return json.RawMessage(resp.GetOkJson()), nil
}

// pickWireProtocol returns forst-node-proto-v1 when both sides support it.
func pickWireProtocol(clientPrefs []string, serverPrefs []string) string {
	clientSet := make(map[string]struct{}, len(clientPrefs))
	for _, p := range clientPrefs {
		clientSet[p] = struct{}{}
	}
	for _, p := range serverPrefs {
		if p == WireProtocolProtoV1 {
			if _, ok := clientSet[WireProtocolProtoV1]; ok {
				return WireProtocolProtoV1
			}
		}
	}
	return ""
}
