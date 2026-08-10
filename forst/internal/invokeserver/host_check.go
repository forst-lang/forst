package invokeserver

import (
	"net/http"
	"strings"
)

var defaultAllowedInvokeHosts = []string{"localhost", "127.0.0.1", "::1"}

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

func requireJSONContentType(r *http.Request) bool {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if ct == "" {
		return false
	}
	return strings.HasPrefix(ct, "application/json")
}
