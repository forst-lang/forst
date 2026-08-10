// host_check validates Host and Content-Type on invoke HTTP requests.
package invokeserver

import (
	"mime"
	"net"
	"net/http"
	"strings"
)

// defaultAllowedInvokeHosts is used when Config.AllowedHosts is nil.
var defaultAllowedInvokeHosts = []string{"localhost", "127.0.0.1", "::1"}

// isAllowedInvokeHost reports whether hostHeader matches allowed (case-insensitive, port stripped).
func isAllowedInvokeHost(hostHeader string, allowed []string) bool {
	host := strings.TrimSpace(hostHeader)
	if host == "" {
		return true
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	host = strings.ToLower(host)
	if allowed == nil {
		allowed = defaultAllowedInvokeHosts
	}
	for _, candidate := range allowed {
		if strings.EqualFold(host, candidate) {
			return true
		}
	}
	return false
}

// requireJSONContentType reports whether r declares an application/json Content-Type.
func requireJSONContentType(r *http.Request) bool {
	ct := strings.TrimSpace(r.Header.Get("Content-Type"))
	if ct == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return strings.EqualFold(mediaType, "application/json")
}
