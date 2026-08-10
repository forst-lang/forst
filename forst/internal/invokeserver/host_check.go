// host_check validates Host and Content-Type on invoke HTTP requests.
package invokeserver

import (
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
	if i := strings.LastIndex(host, ":"); i >= 0 && !strings.HasPrefix(host, "[") {
		if strings.Count(host, ":") == 1 {
			host = host[:i]
		}
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
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if ct == "" {
		return false
	}
	return strings.HasPrefix(ct, "application/json")
}
