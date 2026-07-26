package errorscompat

import (
	"io"
	"strings"
)

// ByteReader implements io.Reader for Forst export-direction interop tests.
type ByteReader struct {
	src string
	off int
}

func NewByteReader(s string) *ByteReader {
	return &ByteReader{src: s}
}

func (r *ByteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.src) {
		return 0, io.EOF
	}
	n := copy(p, r.src[r.off:])
	r.off += n
	err := error(nil)
	if r.off >= len(r.src) {
		err = io.EOF
	}
	return n, err
}

func (r *ByteReader) Remaining() string {
	if r.off >= len(r.src) {
		return ""
	}
	return r.src[r.off:]
}

func Greeting() string {
	return strings.ToUpper("hi")
}
