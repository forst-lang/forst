package hasher

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestWriteHash_matchesEncodingBinaryLittleEndian(t *testing.T) {
	t.Parallel()
	cases := []any{
		uint8(9),
		true,
		false,
		int64(-42),
		uint64(0x0102030405060708),
		NodeHash(0x89abcdef01234567),
		math.Float64frombits(0x3ff0000000000000),
		[]byte("hello"),
		[]byte(nil),
		[]byte{},
	}
	for _, c := range cases {
		var want, got bytes.Buffer
		if err := binary.Write(&want, binary.LittleEndian, c); err != nil {
			t.Fatalf("binary.Write(%T): %v", c, err)
		}
		if err := writeHash(&got, c); err != nil {
			t.Fatalf("writeHash(%T): %v", c, err)
		}
		if !bytes.Equal(want.Bytes(), got.Bytes()) {
			t.Fatalf("writeHash(%T) bytes %x != binary.Write %x", c, got.Bytes(), want.Bytes())
		}
	}
}

func TestWriteHash_stringStillUnsupported(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	if err := writeHash(&buf, "not-fixed-size"); err == nil {
		t.Fatal("expected error for string, matching encoding/binary.Write")
	}
	if buf.Len() != 0 {
		t.Fatal("string must not contribute bytes")
	}
}
