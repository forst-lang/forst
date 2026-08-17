//go:build windows || js

package nodert

import (
	"fmt"
	"os"
)

func dupInvokeAuthHandoffFD(f *os.File) (int, error) {
	if f != nil {
		_ = f.Close()
	}
	return 0, fmt.Errorf("node runtime: in-process invoke auth handoff requires Unix")
}
