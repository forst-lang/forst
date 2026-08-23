package main

import strconv "strconv"

func identity[T any](x T) T {
	return x
}
func main() {
	n := identity(42)
	println(strconv.Itoa(n))
}
