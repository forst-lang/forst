//go:build !windows && !js

package bridgert

import (
	"os"
	"syscall"
)

func dupInvokeAuthHandoffFD(f *os.File) (int, error) {
	if f == nil {
		return 0, bridgeRuntimeErr("invoke auth handoff file is nil")
	}
	writeFD, err := syscall.Dup(int(f.Fd()))
	if err != nil {
		return 0, err
	}
	_ = f.Close()
	return writeFD, nil
}
