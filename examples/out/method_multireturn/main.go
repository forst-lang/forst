package main

import strconv "strconv"

func main() {
	b := NewBuilder()
	s, xs := b.Pair()
	println(s)
	println(strconv.Itoa(len(xs)))
}
