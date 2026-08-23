package main

import fmt "fmt"
import os "os"

type T_BSiWS9EsB18 struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Id       string  `json:"id"`
}

func main() {
	_, resultErr := forst_bridge_callsync_legacy_payment_js_create()
	if !(resultErr == nil) {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", resultErr)
			os.Exit(1)
		}
	}
	println(result.Id)
	println(result.Amount)
}
