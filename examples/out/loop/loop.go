package main

func main() {
	for {
		break
	}
	for false {
		println("unreachable")
	}
	var n int = 0
	for n < 3 {
		n = n + 1
	}
	for i := 0; i < 3; i++ {
		println(string(i))
	}
	for {
		break
	}
	xs := []int{1, 2, 3}
	for range xs {
		println("range")
	}
	for idx := range xs {
		println(string(idx))
	}
	for ik, vv := range xs {
		println(string(string(ik)) + ":" + string(vv))
	}
	for _, elem := range xs {
		println(string(elem))
	}
	for si := range "ab" {
		println(string(si))
	}
	for j := 0; j < 4; j++ {
		if j == 2 {
			continue
		}
		println(string(j))
	}
	var ri int = 0
	var rv int = 0
	for ri, rv = range xs {
		println(string(string(ri)) + "=" + string(rv))
	}
}
