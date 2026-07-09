package main

func main() {
	xs := []int{10, 20, 30, 40}
	mid := xs[1:3]
	tail := xs[2:]
	head := xs[:2]
	println(len(mid))
	println(len(tail))
	println(len(head))
	println(mid[0])
	println(tail[0])
	println(head[1])
}
