package main

import strconv "strconv"

func main() {
	dst := []int{0, 0, 0}
	src := []int{1, 2, 3}
	println(copy(dst, src))
	xs := append(src, 4)
	println(len(xs))
	println(cap(xs))
	println(copy(dst, xs))
	scores := map[string]int{"a": 1, "b": 2}
	empty := map[string]int{}
	println(len(scores))
	println(len(empty))
	for i, v := range xs {
		println(strconv.Itoa(i) + strconv.Itoa(v))
	}
	s := "go"
	p := &s
	println(*p)
	n := 7
	pn := &n
	println(*pn)
}
