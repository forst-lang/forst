// token_file persists the invoke HMAC key at .forst/invoke.token for connect-mode clients.
package invokeserver

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// invokeTokenFileName is the basename written under workDir/.forst/.
const invokeTokenFileName = "invoke.token"

// invokeTokenPath returns workDir/.forst/invoke.token.
func invokeTokenPath(workDir string) string {
	return filepath.Join(workDir, ".forst", invokeTokenFileName)
}

// writeTokenFile atomically writes token to path with mode 0600 (tmp file then rename).
func writeTokenFile(path string, token []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".invoke.token.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	encoded := base64.RawURLEncoding.EncodeToString(token)
	if _, err := tmp.Write([]byte(encoded)); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// readTokenFile loads and base64url-decodes the token at path.
func readTokenFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.RawURLEncoding.DecodeString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("decode invoke token file: %w", err)
	}
	return decoded, nil
}

// removeTokenFile deletes path; a missing file is not an error.
func removeTokenFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
