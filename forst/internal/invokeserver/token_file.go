package invokeserver

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

const invokeTokenFileName = "invoke.token"

func invokeTokenPath(workDir string) string {
	return filepath.Join(workDir, ".forst", invokeTokenFileName)
}

func writeTokenFile(path string, token []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp := path + ".tmp"
	encoded := base64.RawURLEncoding.EncodeToString(token)
	if err := os.WriteFile(tmp, []byte(encoded), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

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

func removeTokenFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
