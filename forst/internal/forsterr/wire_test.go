package forsterr

import (
	"encoding/json"
	"testing"
)

type sampleErr struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

func (e sampleErr) Error() string  { return "taken" }
func (e sampleErr) ForstErrorTag() string { return "CellTaken" }

func TestEncode_nominalError(t *testing.T) {
	w, ok := Encode(sampleErr{Row: 1, Col: 2})
	if !ok {
		t.Fatal("expected ok")
	}
	if w.Tag != "CellTaken" {
		t.Fatalf("tag = %q", w.Tag)
	}
	var payload map[string]int
	if err := json.Unmarshal(w.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["row"] != 1 || payload["col"] != 2 {
		t.Fatalf("payload = %v", payload)
	}
}

func TestEncode_genericError(t *testing.T) {
	_, ok := Encode(errPlain("nope"))
	if ok {
		t.Fatal("expected false for plain error")
	}
}

type errPlain string

func (e errPlain) Error() string { return string(e) }
