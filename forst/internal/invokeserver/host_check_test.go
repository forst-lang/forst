package invokeserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsAllowedInvokeHost_bracketedIPv6WithPort(t *testing.T) {
	if !isAllowedInvokeHost("[::1]:8080", nil) {
		t.Fatal("expected [::1]:8080 to be allowed")
	}
}

func TestRequireJSONContentType_exactMediaType(t *testing.T) {
	okReq := httptest.NewRequest(http.MethodPost, "/invoke", nil)
	okReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	if !requireJSONContentType(okReq) {
		t.Fatal("expected charset parameter to be accepted")
	}

	prefixReq := httptest.NewRequest(http.MethodPost, "/invoke", nil)
	prefixReq.Header.Set("Content-Type", "application/json-seq")
	if requireJSONContentType(prefixReq) {
		t.Fatal("expected prefix-only media type to be rejected")
	}
}
