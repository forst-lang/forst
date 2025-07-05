package main

import errors "errors"

// Password: TypeDefAssertionExpr(TYPE_STRING)
type Password string

func G_H5qf3FeqdN3(password Password) bool {
	if len(password) < 12 {
		return false
	}
	return true
}

func main() {
	var password Password = "12345abc"
	if !G_H5qf3FeqdN3(password) {
		println("Detected password as too weak, exiting...")
		panic(errors.New("assertion failed: " + "Strong()"))
	}
	println("We have a strong password, continuing...")
}
