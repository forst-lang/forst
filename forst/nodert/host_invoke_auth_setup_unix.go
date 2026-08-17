//go:build !windows && !js

package nodert

import (
	"fmt"
	"os"
	"syscall"
)

func dupInvokeAuthHandoffFD(f *os.File) (int, error) {
	if f == nil {
		return 0, fmt.Errorf("node runtime: invoke auth handoff file is nil")
	}
	writeFD, err := syscall.Dup(int(f.Fd()))
	if err != nil {
		return 0, err
	}
	_ = f.Close()
	return writeFD, nil
}
