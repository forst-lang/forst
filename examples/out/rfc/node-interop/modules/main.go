package main

import "strconv"
import fmt "fmt"
import os "os"

type T_Zn4FXrBCht3 struct {
	Id         string  `json:"id"`
	Total      string  `json:"total"`
	TotalCents float64 `json:"totalCents"`
}

func main() {
	order, orderErr := forst_node_callsync_legacy_api_checkout_ts_createOrder()
	if !(orderErr == nil) {
		{
			fmt.Fprintf(os.Stderr, "ensure failed: %v\n", orderErr)
			os.Exit(1)
		}
	}
	println(order.Id)
	println(order.Total)
	println(strconv.FormatFloat(order.TotalCents, 'f', 0, 64))
}
