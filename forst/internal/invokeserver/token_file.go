// token_file persists the invoke HMAC key at .forst/invoke.token for connect-mode clients.
package invokeserver

import (
	"os"
	"path/filepath"
)

// invokeTokenFileName is the basename written under workDir/.forst/.
const invokeTokenFileName = "invoke.token"

// invokeTokenPath returns workDir/.forst/invoke.token.
func invokeTokenPath(workDir string) string {
	return filepath.Join(workDir, ".forst", invokeTokenFileName)
}

// removeTokenFile deletes path; a missing file is not an error.
func removeTokenFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
