package main

import "os"
import fmt "fmt"

func main() {
	if os.Getenv("FORST_INVOKE_ONLY") == "1" {
		println("multipackage-dev invoke-only")
	} else {
		ready, readyErr := forst_node_callsync_host_ts_hostPing()
		if !(readyErr == nil) {
			{
				fmt.Fprintf(os.Stderr, "ensure failed: %v\n", readyErr)
				os.Exit(1)
			}
		}
		println("multipackage-dev ready: " + ready)
	}
	ForstInvokeWaitForShutdown()
}
