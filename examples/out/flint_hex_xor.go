package main

import "fmt"

func flintMask(k []byte) []byte {
	i := 0
	for i < len(k) {
		k[i] = k[i] ^ 54
		i = i + 1
	}
	return k
}
func main() {
	b := []byte("ab")
	fmt.Println(flintMask(b)[0])
	fmt.Println(54)
}
