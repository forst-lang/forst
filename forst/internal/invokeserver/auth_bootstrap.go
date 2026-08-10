package invokeserver

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

const EnvInvokeAuthFD = "FORST_INVOKE_AUTH_FD"

type authHandoffPayload struct {
	Generation uint64 `json:"generation"`
	Token      string `json:"token"`
}

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

func encodeTokenForHandoff(token []byte) string {
	return encodeInvokeProof(token)
}

func decodeTokenFromHandoff(encoded string) ([]byte, error) {
	return decodeInvokeProof(encoded)
}

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
