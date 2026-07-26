package main

func main() {
	var xs [3]int
	xs[0] = 10
	xs[1] = 20
	xs[2] = 30
	println(xs[0])
	println(len(xs))
	var ys []int
	ys = xs[0:2]
	println(len(ys))
}
