package main

import (
	errors "errors"
	"os"
)

// Password: TypeDefAssertionExpr(TYPE_STRING)
type T_gDwvuEDhp1U string

func G_cnwc4DMkTpn(password T_gDwvuEDhp1U) bool {
	return len(password) >= 12
}

func main() {
	var password T_gDwvuEDhp1U = "1234567890123"
	if !G_cnwc4DMkTpn(password) {
		os.Exit(1)
		panic(errors.New("assertion failed: " + "Strong()"))
	}
}
