package main

func main() {
	s := "abcdef"
	mid := s[1:3]
	tail := s[2:]
	head := s[:2]
	println(mid)
	println(tail)
	println(head)
}
