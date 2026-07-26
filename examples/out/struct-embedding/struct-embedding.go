package main

type Inner struct {
	Value int
}

type Outer struct {
	Inner
}

func main() {
	o := Outer{Inner: Inner{Value: 10}}
	println(o.Value)
}
