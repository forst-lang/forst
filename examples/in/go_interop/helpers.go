package main

import "strings"

// GreetUpper is hand-written Go exported for same-package Forst calls.
func GreetUpper(name string) string {
	return strings.ToUpper(name)
}

// AddInts is hand-written Go exported for same-package Forst calls.
func AddInts(a, b int) int {
	return a + b
}
