// Auth bootstrap delivers the live invoke secret to a parent process over an inherited
// file descriptor so spawn-mode clients never read invoke.token from disk.
package invokeserver

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// EnvInvokeAuthFD names the environment variable whose value is the fd number the child
// should write its one-line JSON handoff to (Strong auth profile on Unix spawn).
const EnvInvokeAuthFD = "FORST_INVOKE_AUTH_FD"

// authHandoffPayload is the JSON line written to FORST_INVOKE_AUTH_FD at startup.
type authHandoffPayload struct {
	Generation uint64 `json:"generation"`
	Token      string `json:"token"`
}

// writeAuthHandoff encodes generation and token, writes one JSON line to w, and closes w.
func writeAuthHandoff(w io.WriteCloser, generation uint64, token []byte) error {
	payload := authHandoffPayload{
		Generation: generation,
		Token:      encodeTokenForHandoff(token),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(raw)+"\n"); err != nil {
		return err
	}
	return w.Close()
}

// encodeTokenForHandoff base64url-encodes raw token bytes for the handoff JSON field.
func encodeTokenForHandoff(token []byte) string {
	return encodeInvokeProof(token)
}

// decodeTokenFromHandoff reverses encodeTokenForHandoff.
func decodeTokenFromHandoff(encoded string) ([]byte, error) {
	return decodeInvokeProof(encoded)
}

// openAuthHandoffWriter returns a WriteCloser for the fd named in fdEnv, or ok=false when unset.
func openAuthHandoffWriter(fdEnv string) (io.WriteCloser, bool) {
	raw := os.Getenv(fdEnv)
	if raw == "" {
		return nil, false
	}
	fd, err := strconv.Atoi(raw)
	if err != nil || fd < 0 {
		return nil, false
	}
	f := os.NewFile(uintptr(fd), "invoke-auth-handoff")
	if f == nil {
		return nil, false
	}
	return f, true
}

// readAuthHandoff parses a single JSON handoff line from r (used in tests and tooling).
func readAuthHandoff(r io.Reader) (generation uint64, token []byte, err error) {
	var payload authHandoffPayload
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return 0, nil, fmt.Errorf("read auth handoff: %w", err)
	}
	token, err = decodeTokenFromHandoff(payload.Token)
	if err != nil {
		return 0, nil, err
	}
	return payload.Generation, token, nil
}
