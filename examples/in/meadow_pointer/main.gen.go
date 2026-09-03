package main

import (
	"example.com/meadow_pointer/store"
	"fmt"
)

func main() {
	st := new(store.Store)
	fmt.Println(st.Get())
}
