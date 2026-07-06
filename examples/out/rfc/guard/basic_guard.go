package main

import os "os"
// Password: TypeDefAssertionExpr(String)
type Password string
// T_VzGjSGttgtP: TypeDefShapeExpr({})
type T_VzGjSGttgtP struct {
}
// T_f4j3qrSNtqm: TypeDefShapeExpr({})
type T_f4j3qrSNtqm struct {
}

func G_AAqSwpSKPZ9(password Password) bool {
	if len(string(password)) < 12 {
		return false
	}
	return true
}
func main() {
	var password Password = "12345abc"
	if !G_AAqSwpSKPZ9(password) {
		println("Detected password as too weak, exiting...")
		os.Exit(1)
	}
	println("We have a strong password, continuing...")
}
