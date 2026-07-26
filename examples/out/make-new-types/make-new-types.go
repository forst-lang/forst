package main

func main() {
	xs := make([]int, 10)
	xsCap := make([]int, 10, 20)
	m := make(map[string]int)
	mHint := make(map[string]int, 4)
	p := new(int)
	println(len(xs))
	println(len(xsCap))
	println(len(m))
	println(len(mHint))
	if p != nil {
		println("ok")
	}
}
