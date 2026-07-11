package main

import os "os"

type T_BSiWS9EsB18 struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Id       string  `json:"id"`
}

func main() {
	result, resultErr := forst_node_callsync_legacy_payment_ts_create()
	if !(resultErr == nil) {
		os.Exit(1)
	}
	println(result.Id)
	println(result.Amount)
}
