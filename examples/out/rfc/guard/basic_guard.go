package main

import (
	errors "errors"
)

// Password: TypeDefAssertionExpr(TYPE_STRING)
type T_gDwvuEDhp1U string

func G_H5qf3FeqdN3(password T_gDwvuEDhp1U) bool {
	if len(password) < 12 {
		return false
	}
	return true
}

func main() {
	var password T_gDwvuEDhp1U = "12345abc"
	if !G_H5qf3FeqdN3(password) {
		println("Detected password as too weak, exiting...")
		panic(errors.New("assertion failed: " + "Strong()"))
	}
	println("We have a strong password, continuing...")
}
