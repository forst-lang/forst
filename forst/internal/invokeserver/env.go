// env isolates os.Getenv for tests that stub environment lookups.
package invokeserver

import "os"

// lookupEnv reads key from the process environment.
func lookupEnv(key string) string {
	return os.Getenv(key)
}
