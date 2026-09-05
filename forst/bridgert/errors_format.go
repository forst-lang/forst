package bridgert

import "fmt"

const (
	bridgeRuntimeErrPrefix = "bridge runtime: "
	bridgeHostErrPrefix    = "bridge host: "
)

func bridgeRuntimeErr(format string, args ...any) error {
	return fmt.Errorf(bridgeRuntimeErrPrefix+format, args...)
}

func bridgeHostErr(format string, args ...any) error {
	return fmt.Errorf(bridgeHostErrPrefix+format, args...)
}
