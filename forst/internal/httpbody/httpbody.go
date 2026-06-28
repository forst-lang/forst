// Package httpbody provides helpers to read HTTP request bodies with a byte limit.
package httpbody

import (
	"io"
	"net/http"
	"strings"
)

// DefaultMaxBytes is the default maximum request body size (10MB).
const DefaultMaxBytes = 10 * 1024 * 1024

// normalizeMaxBytes returns maxBytes when positive, otherwise DefaultMaxBytes.
func normalizeMaxBytes(maxBytes int64) int64 {
	if maxBytes <= 0 {
		return DefaultMaxBytes
	}
	return maxBytes
}

// LimitReader wraps r so reads past maxBytes fail.
// maxBytes <= 0 uses DefaultMaxBytes (never unlimited for untrusted HTTP).
func LimitReader(r io.Reader, maxBytes int64) io.Reader {
	rc, ok := r.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(r)
	}
	return http.MaxBytesReader(nil, rc, normalizeMaxBytes(maxBytes))
}

// ReadAll reads from r up to maxBytes.
func ReadAll(r io.Reader, maxBytes int64) ([]byte, error) {
	return io.ReadAll(LimitReader(r, maxBytes))
}

// IsTooLarge reports whether err is from exceeding the MaxBytesReader limit.
func IsTooLarge(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "http: request body too large")
}
