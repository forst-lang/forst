package main

import strconv "strconv"

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
		println(strconv.Itoa(i))
	}
	for {
		break
	}
	xs := []int{1, 2, 3}
	for range xs {
		println("range")
	}
	for idx := range xs {
		println(strconv.Itoa(idx))
	}
	for ik, vv := range xs {
		println(strconv.Itoa(ik) + ":" + strconv.Itoa(vv))
	}
	for _, elem := range xs {
		println(strconv.Itoa(elem))
	}
	for si := range "ab" {
		println(strconv.Itoa(si))
	}
	for j := 0; j < 4; j++ {
		if j == 2 {
			continue
		}
		println(strconv.Itoa(j))
	}
	var ri int = 0
	var rv int = 0
	for ri, rv = range xs {
		println(strconv.Itoa(ri) + "=" + strconv.Itoa(rv))
	}
}
