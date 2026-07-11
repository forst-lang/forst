package main

import os "os"

type T_S47SAU5d2zT struct {
	id string
}

func main() {
	result, resultErr := forst_node_callsync_legacy_payment_ts_create()
	if !(resultErr == nil) {
		os.Exit(1)
	}
	println(result.id)
}
