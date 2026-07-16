package main

import fmt "fmt"
import os "os"

func main() {
	ready, readyErr := forst_node_callsync_host_ts_hostPing()
	if !(readyErr == nil) {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", readyErr)
			os.Exit(1)
		}
	}
	println("multipackage-dev ready: " + ready)
	ForstInvokeWaitForShutdown()
}
