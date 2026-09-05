package hasher

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestWriteHash_matchesEncodingBinaryLittleEndian(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		val  any
	}{
		{name: "uint8", val: uint8(9)},
		{name: "bool_true", val: true},
		{name: "bool_false", val: false},
		{name: "int64", val: int64(-42)},
		{name: "uint64", val: uint64(0x0102030405060708)},
		{name: "NodeHash", val: NodeHash(0x89abcdef01234567)},
		{name: "float64", val: math.Float64frombits(0x3ff0000000000000)},
		{name: "bytes_hello", val: []byte("hello")},
		{name: "bytes_nil", val: []byte(nil)},
		{name: "bytes_empty", val: []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var want, got bytes.Buffer
			if err := binary.Write(&want, binary.LittleEndian, tc.val); err != nil {
				t.Fatalf("binary.Write(%T): %v", tc.val, err)
			}
			if err := writeHash(&got, tc.val); err != nil {
				t.Fatalf("writeHash(%T): %v", tc.val, err)
			}
			if !bytes.Equal(want.Bytes(), got.Bytes()) {
				t.Fatalf("writeHash(%T) bytes %x != binary.Write %x", tc.val, got.Bytes(), want.Bytes())
			}
		})
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
