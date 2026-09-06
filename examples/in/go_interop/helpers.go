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

// StringCounts returns a same-package map type for Forst↔Go composite interop demos.
func StringCounts() map[string]int {
	return map[string]int{"forst": 1}
}

// CountPair returns an unnamed struct for Forst FFI shape-mapping demos.
func CountPair() struct{ N int } {
	return struct{ N int }{N: 1}
}

// Bag is a named same-package Go struct usable from Forst literals and signatures.
type Bag struct {
	Name string
}

// BagName reads an exported field from Bag.
func BagName(b Bag) string { return b.Name }
