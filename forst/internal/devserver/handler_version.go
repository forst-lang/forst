package devserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// jsonMarshalVersionPayload marshals the /version payload; tests may replace to inject errors.
var jsonMarshalVersionPayload = json.Marshal

// handleVersion returns compiler build metadata and the dev HTTP contract version.
func (s *DevServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	payload := map[string]string{
		"version":         s.buildInfo.Version,
		"commit":          s.buildInfo.Commit,
		"date":            s.buildInfo.Date,
		"contractVersion": HTTPContractVersion,
	}
	data, err := jsonMarshalVersionPayload(payload)
	if err != nil {
		s.sendError(w, fmt.Sprintf("failed to marshal version: %v", err), http.StatusInternalServerError)
		return
	}
	response := DevServerResponse{
		Success: true,
		Result:  data,
	}
	s.sendJSONResponse(w, response)
}
