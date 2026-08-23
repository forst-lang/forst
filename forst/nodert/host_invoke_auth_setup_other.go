//go:build windows || js

package nodert

import (
	"os"
)

func dupInvokeAuthHandoffFD(f *os.File) (int, error) {
	if f != nil {
		_ = f.Close()
	}
	return 0, bridgeRuntimeErr("in-process invoke auth handoff requires Unix")
}
